package selfupgrade

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MaxBinary caps a download.
//
// A ceiling rather than a guess at the real size: the agent is around twenty
// megabytes, and without a limit a redirect to somewhere unexpected — or a
// control plane serving the SPA's HTML because a path was wrong — streams into
// memory until the host runs out. That failure is a hypervisor going down, not
// an upgrade that did not happen.
const MaxBinary = 256 << 20

// maxBinaryForTest is the cap actually applied. A variable so the ceiling can be
// exercised without allocating a quarter of a gigabyte in a unit test — the
// behaviour under test is the refusal, not the number.
var maxBinaryForTest int64 = MaxBinary

// Download fetches the agent binary for one platform.
//
// A plain HTTP GET rather than the generated client. The endpoint returns raw
// executable bytes, and the generator models that body as a Go string on the
// assumption it is text — which is fine for Go strings but routes twenty
// megabytes through a decoder built for JSON. Streaming it here also means the
// response is read once, into the buffer that is about to be hashed.
func Download(
	ctx context.Context, httpc *http.Client, baseURL, keyPublic, keySecret, goos, goarch string,
) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v1/agent/dist/%s/%s",
		strings.TrimRight(baseURL, "/"), goos, goarch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("upgrade: build the download request: %w", err)
	}
	req.Header.Set("X-AGENT-Key", keyPublic)
	req.Header.Set("X-AGENT-Secret", keySecret)
	req.Header.Set("Accept", "application/octet-stream")

	res, err := httpc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upgrade: download the agent: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		// The first part of the body, not the whole of it: an error response is
		// small, and an unexpected 200-shaped HTML page is not something to quote
		// in full into a log.
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 200))
		return nil, fmt.Errorf("upgrade: the control plane answered %s: %s",
			res.Status, strings.TrimSpace(string(snippet)))
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxBinaryForTest+1))
	if err != nil {
		return nil, fmt.Errorf("upgrade: read the download: %w", err)
	}
	if int64(len(body)) > maxBinaryForTest {
		return nil, fmt.Errorf("upgrade: the download exceeded %d bytes, which is not an agent "+
			"binary; refusing to hold it in memory on a hypervisor", maxBinaryForTest)
	}
	return body, nil
}
