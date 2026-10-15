package fleetops

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CorosyncReload repairs a member whose RUNNING corosync is behind its own
// configuration file.
//
// pmxcfs writes /etc/corosync/corosync.conf when the cluster's copy changes and
// then asks corosync to reload it. That request can be lost. Observed live: two
// nodes added to one cluster seven seconds apart, and the second addition's
// reload reached a member still settling from the first. Its file said version
// 14 and its corosync ran 13, which had no entry for the newest node — so the
// two had no link, the newcomer was never quorate, and nothing anywhere said
// why. Every read the agent makes of /etc/pve looked healthy on the stale
// member, because the stale part is corosync's memory, not a file.
//
// The repair is the one an operator ran by hand: corosync-cfgtool -R. Reloading
// a file that is already loaded changes nothing, so a redundant reload is
// harmless — but the drift must PERSIST for driftGrace first, because pmxcfs
// writes the file a moment before it triggers its own reload and a check landing
// in that window would race it.
//
// Only a runtime BEHIND the file is acted on. A runtime ahead of it is a file
// pmxcfs has not rewritten yet, and reloading would load the older one.
type CorosyncReload struct {
	Runner Runner

	mu          sync.Mutex
	behindSince time.Time
	lastReload  time.Time
	now         func() time.Time
}

// Vars rather than consts so a test can shorten them.
var (
	// driftGrace is how long the runtime must stay behind before reloading.
	driftGrace = 30 * time.Second
	// reloadInterval spaces repeated reloads of a drift that did not clear, so a
	// reload corosync keeps refusing is retried rather than hammered.
	reloadInterval = time.Minute
)

var configVersionRE = regexp.MustCompile(`(?m)^\s*config_version\s*:\s*(\d+)\s*$`)

// fileConfigVersion is the config_version in this host's corosync.conf, and
// false when the host has none — a standalone host, which has nothing to repair.
func fileConfigVersion() (uint64, bool, error) {
	b, err := os.ReadFile(CorosyncConf)
	if os.IsNotExist(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	m := configVersionRE.FindSubmatch(b)
	if m == nil {
		return 0, false, fmt.Errorf("%s carries no config_version", CorosyncConf)
	}
	v, err := strconv.ParseUint(string(m[1]), 10, 64)
	return v, err == nil, err
}

// runtimeConfigVersion asks the running corosync which configuration it loaded.
// Its output is "totem.config_version (u64) = 14".
func (c *CorosyncReload) runtimeConfigVersion(ctx context.Context) (uint64, error) {
	out := newOutputTail(4)
	if err := c.Runner.Run(ctx, out.log, "corosync-cmapctl", "-g", "totem.config_version"); err != nil {
		return 0, fmt.Errorf("corosync-cmapctl: %w%s", err, out.suffix())
	}
	out.mu.Lock()
	lines := append([]string(nil), out.lines...)
	out.mu.Unlock()
	for _, l := range lines {
		if i := strings.LastIndex(l, "="); i >= 0 {
			if v, err := strconv.ParseUint(strings.TrimSpace(l[i+1:]), 10, 64); err == nil {
				return v, nil
			}
		}
	}
	return 0, fmt.Errorf("corosync-cmapctl printed no config_version%s", out.suffix())
}

// Check compares the running configuration with the file and reloads when the
// runtime has stayed behind for longer than driftGrace.
//
// Never fatal and never blocking for long: it runs on every report cycle, and a
// corosync that is down or mid-restart is ordinary on a cluster host — that is
// pve-cluster's to report, not this.
func (c *CorosyncReload) Check(ctx context.Context, logf func(string, ...any)) {
	if c == nil || c.Runner == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now
	if c.now != nil {
		now = c.now
	}

	file, ok, err := fileConfigVersion()
	if err != nil || !ok {
		c.behindSince = time.Time{}
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	running, err := c.runtimeConfigVersion(cctx)
	if err != nil || running >= file {
		c.behindSince = time.Time{}
		return
	}

	t := now()
	if c.behindSince.IsZero() {
		c.behindSince = t
		return
	}
	if t.Sub(c.behindSince) < driftGrace || t.Sub(c.lastReload) < reloadInterval {
		return
	}
	c.lastReload = t
	logf("corosync is running configuration version %d but %s is version %d, and has been "+
		"for %s; reloading it", running, CorosyncConf, file, t.Sub(c.behindSince).Round(time.Second))
	out := newOutputTail(10)
	if err := c.Runner.Run(cctx, out.log, "corosync-cfgtool", "-R"); err != nil {
		logf("corosync-cfgtool -R failed, retrying in %s: %v%s", reloadInterval, err, out.suffix())
	}
}
