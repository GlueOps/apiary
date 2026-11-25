# Running the agent under systemd

    install -m 0755 apiary-agent /usr/sbin/apiary-agent
    install -m 0644 apiary-agent.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable --now apiary-agent

The unit is `Type=notify` with a watchdog, so `apiary-agent start` must tell
systemd it is ready and keep checking in. Running the binary by hand outside
systemd is still fine — the notification is skipped when `NOTIFY_SOCKET` is
absent.

There is deliberately no timer. The agent is long-running because its heartbeat
is what tells the control plane a host is alive, and a periodic invocation would
turn every gap between runs into an apparent outage.

## Package steps run in their own units

`apt-get` and `dpkg` do not run inside `apiary-agent.service`. Each step of a
dist-upgrade (`apt-get update`, `apt-get dist-upgrade`, and the repairs) is
started with `systemd-run` as a transient unit named
`apiary-step-<task id>-<step>.service`, so restarting, upgrading or
watchdog-killing the agent never kills dpkg part-way through.

- See one in flight: `systemctl list-units 'apiary-step-*'`, and
  `journalctl -u 'apiary-step-*'`.
- Each step's record is in `/var/lib/apiary-agent/steps/`: `<key>.started`,
  `<key>.log` (stdout and stderr), and `<key>.rc` (the exit status, written only
  when the step ends). A restarted agent reads these to re-attach to a running
  step or reuse a finished one, never to start a second apt. Records older than
  30 days are pruned when the agent starts.
- Do not stop an `apiary-step-*` unit by hand unless you mean to interrupt dpkg.
  A step with a `.started` file, no `.rc` and no unit is treated as killed: the
  agent repairs the package database (`dpkg --configure -a`, then
  `apt-get -f install --no-remove`) before running anything else, and stops for
  a person if that does not leave dpkg consistent.

The units are siblings of the agent and get systemd's default sandbox, not the
agent unit's hardening; see the comments in `apiary-agent.service`.
