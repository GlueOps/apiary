# Running the agent under systemd

    install -m 0755 apiary-agent /usr/local/bin/apiary-agent
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
