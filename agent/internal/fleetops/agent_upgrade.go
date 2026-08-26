package fleetops

// KindAgentUpgrade replaces the agent's own binary with the one the control
// plane is serving.
//
// It is the only task whose successful completion kills the process running it,
// and that shapes the whole implementation — see the executor in
// cmd/agent/upgrade_task.go.
const KindAgentUpgrade = "fleet.host.agent-upgrade"
