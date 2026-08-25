package proxmox

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RetryOption func(*retryPolicy)

type retryPolicy struct {
	maxAttempts    int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	condition      func(*http.Response, error) bool

	now   func() time.Time
	sleep func(context.Context, time.Duration) error

	mu   sync.Mutex
	rand *rand.Rand
}

const (
	defaultRetryMaxAttempts    = 3
	defaultRetryInitialBackoff = 200 * time.Millisecond
	defaultRetryMaxBackoff     = 5 * time.Second
)

func defaultRetryPolicy() *retryPolicy {
	return &retryPolicy{
		maxAttempts:    defaultRetryMaxAttempts,
		initialBackoff: defaultRetryInitialBackoff,
		maxBackoff:     defaultRetryMaxBackoff,
		condition:      defaultRetryCondition,
		now:            time.Now,
		sleep:          contextSleep,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())), //nolint:gosec // backoff jitter is not security-sensitive
	}
}

func defaultRetryCondition(res *http.Response, err error) bool {
	if err != nil {

		return true
	}
	if res == nil {
		return false
	}
	switch res.StatusCode {
	case http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusTooManyRequests:
		return true
	}
	return false
}

func WithRetryMax(n int) RetryOption {
	return func(p *retryPolicy) {
		if n < 1 {
			n = 1
		}
		p.maxAttempts = n
	}
}

func WithRetryBackoff(initial, max time.Duration) RetryOption {
	return func(p *retryPolicy) {
		if initial > 0 {
			p.initialBackoff = initial
		}
		if max > 0 {
			p.maxBackoff = max
		}
	}
}

func WithRetryCondition(fn func(*http.Response, error) bool) RetryOption {
	return func(p *retryPolicy) {
		if fn != nil {
			p.condition = fn
		}
	}
}

func WithRetry(opts ...RetryOption) Option {
	policy := defaultRetryPolicy()
	for _, o := range opts {
		o(policy)
	}
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

func (c *Client) installRetryWrapper() {
	if c.retryPolicy == nil {
		return
	}
	c.ensureOwnHTTPClient()
	base := c.httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	c.httpClient.Transport = &retryRoundTripper{
		base:   base,
		policy: c.retryPolicy,
	}
}

type retryRoundTripper struct {
	base   http.RoundTripper
	policy *retryPolicy
}

func (r *retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rewind, err := snapshotBody(req)
	if err != nil {
		return nil, err
	}

	idempotent := isIdempotent(req.Method) || rewind != nil

	var (
		lastRes *http.Response
		lastErr error
	)

	for attempt := 0; attempt < r.policy.maxAttempts; attempt++ {

		if attempt > 0 && rewind != nil {
			req.Body = rewind()
		}

		res, err := r.base.RoundTrip(req)

		if err != nil && ctxErr(ctx) != nil {
			return nil, ctx.Err()
		}

		lastRes, lastErr = res, err

		if !r.policy.condition(res, err) {
			return res, err
		}

		if !idempotent {
			return res, err
		}

		if attempt == r.policy.maxAttempts-1 {
			return res, err
		}

		delay := r.computeBackoff(attempt, res)

		drainResponse(res)

		if err := r.policy.sleep(ctx, delay); err != nil {
			return nil, err
		}
	}

	return lastRes, lastErr
}

func (r *retryRoundTripper) computeBackoff(attempt int, res *http.Response) time.Duration {
	if res != nil {
		if d, ok := parseRetryAfter(res.Header.Get("Retry-After"), r.policy.now()); ok {
			if d > r.policy.maxBackoff {
				d = r.policy.maxBackoff
			}
			if d < 0 {
				d = 0
			}
			return d
		}
	}

	window := r.policy.initialBackoff
	for i := 0; i < attempt && window < r.policy.maxBackoff; i++ {
		window *= 2
	}
	if window > r.policy.maxBackoff {
		window = r.policy.maxBackoff
	}
	if window <= 0 {
		return 0
	}

	r.policy.mu.Lock()
	defer r.policy.mu.Unlock()
	return time.Duration(r.policy.rand.Int63n(int64(window)))
}

func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete,
		http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}

func snapshotBody(req *http.Request) (func() io.ReadCloser, error) {
	if req.GetBody != nil {
		return func() io.ReadCloser {
			rc, err := req.GetBody()
			if err != nil {
				return io.NopCloser(strings.NewReader(""))
			}
			return rc
		}, nil
	}
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	buf, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(buf))

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), nil
	}
	return func() io.ReadCloser {
		return io.NopCloser(bytes.NewReader(buf))
	}, nil
}

func drainResponse(res *http.Response) {
	if res == nil || res.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
}

func ctxErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func contextSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctxErr(ctx)
	}
	if ctx == nil {
		time.Sleep(d)
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
