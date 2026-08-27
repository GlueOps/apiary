// Package bee is the fleet agent's runtime: two loops over one local database.
//
// The split is the design, not an implementation detail.
//
//	report loop   owns ALL network I/O — heartbeat, claim, ship logs, report
//	execute loop  owns ALL local work  — never touches the network
//
// They meet only through the store. That is what lets a bee keep heartbeating
// through a twenty-minute `apt upgrade`: the loop doing the upgrading is not the
// loop answering "are you alive". Every other arrangement — one loop, or two
// that share a client — ends with a long task starving liveness, and a control
// plane concluding a working hypervisor is dead in the middle of a rotation.
//
// It also makes crash recovery ordinary. The store is the source of truth, both
// loops read it on start, and neither reconstructs anything from memory that did
// not survive the reboot it just performed.
package bee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/glueops/apiary-agent/internal/firewall"
	"github.com/glueops/apiary-agent/internal/selfupgrade"
	"github.com/glueops/apiary-agent/internal/store"
	apiary "github.com/glueops/apiary/sdk/go"
)

// Executor performs one kind of task. It receives an Env whose logging goes to
// the LOCAL outbox — an executor never speaks to the control plane, so a network
// outage mid-upgrade can neither block nor fail the upgrade.
type Executor func(ctx context.Context, t store.Task, e *Env) error

// Env is what an executor may do besides its own work.
type Env struct {
	// Logf appends a line to the outbox; the report loop ships it.
	Logf func(stream, format string, args ...any)
	// Phase records what is happening now, so an operator watching a rotation
	// sees progress rather than a spinner.
	Phase func(phase string)
	// Declare announces, BEFORE the interrupting action, how long this host
	// expects to be unreachable. Synchronous, and it returns an error if the
	// control plane did not acknowledge: an executor about to reboot must not
	// proceed believing the declaration landed, because the entire point is that
	// the control plane knows before the silence starts.
	Declare func(ctx context.Context, d time.Duration, reason string) error
}

type Options struct {
	Store        *store.Store
	Client       *apiary.APIClient
	AgentVersion string

	HeartbeatWait time.Duration
	IdleBackoff   time.Duration
	LogBatch      int

	// FirewallRunner executes the commands that load a ruleset. A dependency so
	// the convergence path can be driven in a test without nft and without the
	// ability to break the machine running it — which matters more here than
	// anywhere else in this agent, because the failure mode is an unreachable
	// host rather than a failed task.
	FirewallRunner firewall.Runner
}

type Bee struct {
	opts      Options
	executors map[string]Executor
	notify    *notifier

	// firewall is what this host is enforcing, as established by this process.
	firewall appliedFirewall

	// desiredFirewall is what the last heartbeat said it should be. Nil means
	// hold.
	desiredFirewall atomic.Value

	// sendDigest is cleared the first time a control plane rejects the agent
	// digest as an unknown field, which is how an older server answers a newer
	// agent. Starts true.
	sendDigest atomic.Bool

	// declareMu serialises Declare against the report loop's own heartbeat, so a
	// declaration cannot be lost to a heartbeat that raced it.
	declareMu sync.Mutex
}

func New(opts Options, executors map[string]Executor) *Bee {
	if opts.HeartbeatWait <= 0 {
		opts.HeartbeatWait = 25 * time.Second
	}
	if opts.FirewallRunner == nil {
		opts.FirewallRunner = firewall.OSRunner{}
	}
	if opts.IdleBackoff <= 0 {
		opts.IdleBackoff = 2 * time.Second
	}
	if opts.LogBatch <= 0 {
		opts.LogBatch = 200
	}
	b := &Bee{opts: opts, executors: executors, notify: newNotifier()}
	// Sent until a control plane says it does not know the field. See the 422
	// handling in heartbeatLocked.
	b.sendDigest.Store(true)
	return b
}

func (b *Bee) Run(ctx context.Context) error {
	// Both loops stamp liveness before anything else, so the watchdog does not
	// see them as stale during start-up and restart the agent while it is coming
	// up — which would be a boot loop that looks like a crash loop.
	now := time.Now().UnixNano()
	b.notify.reportAlive.Store(now)
	b.notify.executeAlive.Store(now)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); b.reportLoop(ctx) }()
	go func() { defer wg.Done(); b.executeLoop(ctx) }()
	go func() { defer wg.Done(); b.notify.Watchdog(ctx) }()

	b.notify.Ready()
	b.notify.Status("waiting for work")

	wg.Wait()
	b.notify.Stopping()
	return ctx.Err()
}

func (b *Bee) reportLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		b.notify.reportAlive.Store(time.Now().UnixNano())

		// Logs before status: a task that has just failed should have its output
		// on the server before its outcome, so whoever sees "failed" can already
		// read why.
		if err := b.shipLogs(ctx); err != nil {
			slog.WarnContext(ctx, "could not ship logs", "error", err)
		}
		if err := b.reportTerminal(ctx); err != nil {
			slog.WarnContext(ctx, "could not report task outcome", "error", err)
		}

		if _, err := b.heartbeat(ctx, heartbeatOpts{wait: b.opts.HeartbeatWait}); err != nil {
			slog.WarnContext(ctx, "heartbeat failed", "error", err)
		}

		// AFTER the heartbeat returns and its lock is released. Convergence
		// confirms itself with a heartbeat of its own, and running it inside the
		// one that delivered the instruction would deadlock on declareMu.
		if d, _ := b.desiredFirewall.Load().(*desiredFirewall); d != nil {
			b.converge(ctx, d.revision, d.digest)
		}

		// ALWAYS pause, whether or not a task came back.
		//
		// The obvious version — loop immediately when a task arrives — spins at
		// full speed, because the control plane hands a bee the task it ALREADY
		// HOLDS on every heartbeat. That re-delivery is deliberate (it is how a
		// crashed bee resumes), so "a task came back" is not the same as "there
		// is new work", and treating it that way had one agent exhaust a 200/min
		// rate limit within a minute of picking up its first task.
		//
		// Latency does not suffer: the long poll is what makes pickup prompt,
		// and the execute loop is already watching the store independently.
		if !sleep(ctx, b.opts.IdleBackoff) {
			return
		}
	}
}

type heartbeatOpts struct {
	wait              time.Duration
	unavailableFor    time.Duration
	unavailableReason string
	back              bool
}

func (b *Bee) heartbeat(ctx context.Context, o heartbeatOpts) (bool, error) {
	b.declareMu.Lock()
	defer b.declareMu.Unlock()
	return b.heartbeatLocked(ctx, o)
}

// heartbeatLocked is heartbeat with declareMu already held, so the 422 retry
// above can re-enter without deadlocking on a lock it is already inside.
func (b *Bee) heartbeatLocked(ctx context.Context, o heartbeatOpts) (bool, error) {

	body := *apiary.NewHeartbeatRequestBody()
	body.SetAgentVersion(b.opts.AgentVersion)
	// The digest of the binary actually running. The control plane decides
	// whether this host needs upgrading on this rather than on the version,
	// because a version is a label somebody stamped and only a release stamps the
	// server and the binaries it serves from one tag.
	//
	// Withheld once a control plane has rejected it — see sendDigest.
	if b.sendDigest.Load() {
		if d, err := selfupgrade.SelfDigest(); err == nil {
			body.SetAgentDigest(d)
		}
	}
	body.SetWaitSeconds(int64(o.wait.Seconds()))
	// Reported every time, so convergence is a fact the control plane observes
	// rather than one it assumes from having published.
	if rev, digest := b.firewall.get(); rev > 0 && digest != "" {
		body.SetAppliedFirewallRevision(int64(rev))
		body.SetAppliedFirewallDigest(digest)
	}
	// What this host is actually running, counted from /etc/pve.
	//
	// The control plane records it only while this machine is in no cluster,
	// which is the gap the per-cluster capacity sync structurally cannot see —
	// and the state a machine is in when it is about to join one, where a wrong
	// zero means the arrival gate waves through a join that discards guests.
	//
	// A failure to count is silence, not zero. Sending zero because a read
	// failed would assert an observation nobody made.
	if n, err := CountLocalGuests(); err == nil {
		body.SetObservedGuests(int64(n))
	}

	if o.back {
		body.SetBack(true)
	} else if o.unavailableFor > 0 {
		body.SetUnavailableForSeconds(int64(o.unavailableFor.Seconds()))
		body.SetUnavailableReason(o.unavailableReason)
	}

	res, httpRes, err := b.opts.Client.FleetAPI.FleetBeeHeartbeat(ctx).HeartbeatRequestBody(body).Execute()
	if err != nil {
		// A 422 means the control plane refused the BODY, and the only field this
		// agent sends that an older one may not know is the digest. huma rejects an
		// unknown body field before the handler runs, so an agent newer than its
		// control plane would otherwise fail every heartbeat and look offline while
		// being perfectly healthy.
		//
		// Self-upgrade normally makes this impossible — the binary comes FROM the
		// control plane, so an agent cannot be newer than the server that served it
		// — but a rolling deploy has both versions live at once, and a
		// hand-installed agent has no such guarantee at all.
		//
		// Stopped rather than retried per request: once a control plane has said it
		// does not know the field, sending it again only produces the same 422.
		if httpRes != nil && httpRes.StatusCode == http.StatusUnprocessableEntity &&
			b.sendDigest.Swap(false) {
			slog.WarnContext(ctx, "the control plane rejected the agent digest; "+
				"it is older than this agent. Continuing without it — upgrades will be "+
				"decided on the version instead, which cannot converge if the two disagree")
			return b.heartbeatLocked(ctx, o)
		}
		return false, err
	}
	// Recorded, not acted on here. Converging takes seconds and can roll back with
	// its own heartbeat, and doing that while holding declareMu would deadlock
	// against the confirm — so the report loop acts on it once the lock is gone.
	if fw, ok := res.GetFirewallOk(); ok && fw != nil {
		b.desiredFirewall.Store(&desiredFirewall{
			revision: int(fw.GetRevision()), digest: fw.GetDigest(),
		})
	} else {
		// Absent means HOLD. Cleared rather than left standing, so a machine that
		// leaves its fleet stops being told to converge on that fleet's policy.
		b.desiredFirewall.Store((*desiredFirewall)(nil))
	}

	task, ok := res.GetTaskOk()
	if !ok || task == nil {
		return false, nil
	}

	spec := "{}"
	if raw, err := json.Marshal(task.GetSpec()); err == nil {
		spec = string(raw)
	}
	// Written before anything acts on it: a bee that dies between accepting and
	// starting still knows on restart that this task is its responsibility.
	if err := b.opts.Store.AcceptTask(ctx, store.Task{
		ID: task.GetId(), Kind: task.GetKind(), Spec: spec, Attempt: int(task.GetAttempt()),
	}); err != nil {
		return false, fmt.Errorf("accept task locally: %w", err)
	}
	return true, nil
}

func (b *Bee) shipLogs(ctx context.Context) error {
	lines, err := b.opts.Store.UnshippedLogs(ctx, b.opts.LogBatch)
	if err != nil || len(lines) == 0 {
		return err
	}

	byTask := map[string][]store.LogLine{}
	var order []string
	for _, l := range lines {
		if _, seen := byTask[l.TaskID]; !seen {
			order = append(order, l.TaskID)
		}
		byTask[l.TaskID] = append(byTask[l.TaskID], l)
	}

	for _, taskID := range order {
		batch := byTask[taskID]
		payload := make([]apiary.TaskLogLine, 0, len(batch))
		var highest int64
		for _, l := range batch {
			line := *apiary.NewTaskLogLine(l.Message, l.Seq)
			line.SetStream(l.Stream)
			line.SetLoggedAt(l.LoggedAt)
			payload = append(payload, line)
			if l.Seq > highest {
				highest = l.Seq
			}
		}
		body := *apiary.NewLogsRequestBody(payload)

		if _, _, err := b.opts.Client.FleetAPI.FleetBeeAppendLogs(ctx, taskID).
			LogsRequestBody(body).Execute(); err != nil {
			// Deliberately not marked shipped. The batch is resent, and the
			// server's (task_id, seq) uniqueness makes the resend a no-op — which
			// is what makes pessimism free here. Marking optimistically would
			// lose the tail of a failed rotation on any blip, precisely when the
			// log is the only evidence of what went wrong.
			return fmt.Errorf("ship logs for task %s: %w", taskID, err)
		}
		if err := b.opts.Store.MarkShipped(ctx, highest); err != nil {
			return err
		}
	}
	return b.opts.Store.PruneShipped(ctx, 5000)
}

func (b *Bee) reportTerminal(ctx context.Context) error {
	t, err := b.opts.Store.CurrentTask(ctx)
	if err != nil || t == nil || t.Reported {
		return err
	}
	if t.Status != store.TaskDone && t.Status != store.TaskFailed {
		return nil
	}

	status := "succeeded"
	if t.Status == store.TaskFailed {
		status = "failed"
	}
	body := *apiary.NewReportRequestBody(status)
	body.SetPhase(t.Phase)
	if t.Error != "" {
		body.SetError(t.Error)
	}

	if _, _, err := b.opts.Client.FleetAPI.FleetBeeReportTask(ctx, t.ID).
		ReportRequestBody(body).Execute(); err != nil {
		return err
	}
	if err := b.opts.Store.MarkReported(ctx, t.ID); err != nil {
		return err
	}
	// Only once the outcome is acknowledged AND every line has shipped;
	// ClearTask enforces the second half.
	return b.opts.Store.ClearTask(ctx, t.ID)
}

func (b *Bee) executeLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		b.notify.executeAlive.Store(time.Now().UnixNano())
		t, err := b.opts.Store.CurrentTask(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "could not read the local task", "error", err)
		}
		if t == nil || t.Status == store.TaskDone || t.Status == store.TaskFailed {
			if !sleep(ctx, time.Second) {
				return
			}
			continue
		}
		b.execute(ctx, *t)
	}
}

func (b *Bee) execute(ctx context.Context, t store.Task) {
	env := &Env{
		Logf: func(stream, format string, args ...any) {
			_ = b.opts.Store.AppendLog(ctx, t.ID, stream, fmt.Sprintf(format, args...))
		},
		Phase: func(phase string) { _ = b.opts.Store.SetPhase(ctx, t.ID, phase) },
		Declare: func(ctx context.Context, d time.Duration, reason string) error {
			// wait 0: the declaration must not be delayed behind a long poll.
			_, err := b.heartbeat(ctx, heartbeatOpts{unavailableFor: d, unavailableReason: reason})
			return err
		},
	}

	exec, ok := b.executors[t.Kind]
	if !ok {
		env.Logf("agent", "no executor is registered for task kind %q", t.Kind)
		_ = b.opts.Store.Finish(ctx, t.ID, store.TaskFailed,
			fmt.Sprintf("this agent cannot perform %q; it may be older than the control plane", t.Kind))
		return
	}

	if err := b.opts.Store.MarkRunning(ctx, t.ID, "starting"); err != nil {
		slog.ErrorContext(ctx, "could not mark the task running", "error", err)
		return
	}
	env.Logf("agent", "starting %s (attempt %d)", t.Kind, t.Attempt)
	b.notify.Status("running %s", t.Kind)
	defer b.notify.Status("waiting for work")

	if err := runExecutor(ctx, exec, t, env); err != nil {
		if errors.Is(err, context.Canceled) {
			// Shutting down, not failing. Leaving the task RUNNING locally means
			// a restarted bee resumes it rather than reporting a failure that
			// never happened.
			env.Logf("agent", "interrupted; will resume on restart")
			return
		}
		env.Logf("agent", "failed: %v", err)
		_ = b.opts.Store.Finish(ctx, t.ID, store.TaskFailed, err.Error())
		return
	}
	env.Logf("agent", "completed %s", t.Kind)
	_ = b.opts.Store.Finish(ctx, t.ID, store.TaskDone, "")
}

// Back withdraws a standing unavailability declaration. Called once at start-up:
// a bee that has come up after a declared reboot is, by definition, back.
func (b *Bee) Back(ctx context.Context) error {
	_, err := b.heartbeat(ctx, heartbeatOpts{back: true})
	return err
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// runExecutor runs one executor and turns a panic into a failed task.
//
// WITHOUT THIS, ONE BAD TASK TAKES THE HOST DOWN PERMANENTLY. A panic in an
// executor kills the process; systemd restarts it — Restart=always with no rate
// limit, deliberately — the agent collects the SAME task, and panics again. The
// host is then in a crash loop that no amount of waiting resolves, and the only
// evidence is a stack trace in the journal of a machine nobody is looking at.
//
// That is not hypothetical. A dist-upgrade task reached attempt 125 against a
// host whose Proxmox client was nil, panicking on every one, for hours.
//
// The blast radius is what makes it worth a recover: tasks are queued per bee,
// so a fleet-wide operation that panics takes down EVERY agent in the fleet the
// same way. A failed task is recoverable and visible; a fleet of crash-looping
// agents is neither.
//
// The panic is reported as the task's error, so it lands in the control plane
// where somebody will see it rather than in a journal where nobody will.
func runExecutor(ctx context.Context, exec Executor, t store.Task, env *Env) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// The stack goes to the log, which is where a stack belongs; the task
			// carries the short form, because it is rendered beside the machine.
			slog.ErrorContext(ctx, "task executor panicked",
				"kind", t.Kind, "task_id", t.ID, "panic", r,
				"stack", string(debug.Stack()))
			err = fmt.Errorf("the agent panicked running %s: %v", t.Kind, r)
		}
	}()
	return exec(ctx, t, env)
}
