package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/glueops/apiary-agent/internal/bee"
	"github.com/glueops/apiary-agent/internal/fleetops"
	"github.com/glueops/apiary-agent/internal/store"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the fleet agent",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		st, err := store.Open(ctx, cfg.StatePath)
		if err != nil {
			return err
		}
		defer func() { _ = st.Close() }()

		id, err := st.Identity(ctx)
		if errors.Is(err, store.ErrNotEnrolled) {
			return fmt.Errorf("this host has not enrolled yet; run `apiary-agent enrol --token <token>`")
		}
		if err != nil {
			return err
		}

		client, err := bee.NewClient(id.BaseURL, id.KeyPublic, id.KeySecret)
		if err != nil {
			return err
		}

		// Built ONCE and shared between the task executors and the convergence
		// loop, so the two cannot end up talking to the hypervisor through
		// differently-configured clients — which is the sort of difference that
		// only shows up as one of them mysteriously failing to authenticate.
		fd := fleetDeps(cfg)

		b := bee.New(bee.Options{
			Store:         st,
			Client:        client,
			AgentVersion:  Version,
			HeartbeatWait: cfg.HeartbeatWait,
			Fleet:         fd,
		}, executors(fd, cfg))

		// Withdraw any standing unavailability BEFORE the loops start. This
		// process may be the one that came back from the reboot it declared, and
		// until it says so the control plane is still holding that host as
		// expectedly-absent — which suppresses exactly the alerting that should
		// resume now.
		if err := b.Back(ctx); err != nil {
			slog.WarnContext(ctx, "could not report that this host is back; continuing", "error", err)
		}

		slog.InfoContext(ctx, "fleet agent started",
			"bee_id", id.BeeID, "org_id", id.OrgID, "base_url", id.BaseURL, "version", Version)

		return b.Run(ctx)
	},
}

// executors is the registry of task kinds this build can perform.
//
// A kind that is absent is reported as a failure naming the version, rather than
// ignored: a task that silently does nothing on an old agent is far worse during
// a rotation than one that says "I cannot do this".
//
// Each fleet operation is a thin wrapper over internal/fleetops. The wrappers
// exist to translate bee.Env into fleetops.Env and nothing else — the decisions,
// which are the dangerous part, live in fleetops where they are tested against a
// fake cluster rather than a hypervisor.
func executors(d fleetops.Deps, cfg Config) map[string]bee.Executor {
	adapt := func(e *bee.Env) *fleetops.Env {
		return &fleetops.Env{Logf: e.Logf, Phase: e.Phase, Declare: e.Declare}
	}
	return map[string]bee.Executor{
		"fleet.host.ping": func(_ context.Context, _ store.Task, e *bee.Env) error {
			e.Phase("pinging")
			e.Logf("agent", "pong from the fleet agent")
			return nil
		},
		fleetops.KindDistUpgrade: func(ctx context.Context, t store.Task, e *bee.Env) error {
			// t.Phase carries what this task had already reached before the host
			// rebooted; DistUpgrade uses it to resume rather than start over.
			return fleetops.DistUpgrade(ctx, d, adapt(e), t.Phase, t.Spec)
		},
		fleetops.KindJoin: func(ctx context.Context, t store.Task, e *bee.Env) error {
			return fleetops.Join(ctx, d, adapt(e), t.Spec)
		},
		fleetops.KindEvict: func(ctx context.Context, t store.Task, e *bee.Env) error {
			return fleetops.Evict(ctx, d, adapt(e), t.Spec)
		},
		// Not routed through fleetops: it touches no hypervisor and needs the
		// agent's own configuration and credential, which fleetops.Deps has no
		// business carrying.
		fleetops.KindAgentUpgrade: upgradeTask(cfg),
	}
}

// fleetDeps wires the executors to the local hypervisor.
//
// A host that is not a Proxmox node still runs the agent perfectly well — it
// just cannot perform the fleet tasks, and says so when handed one, rather than
// refusing to start. An agent that would not boot without a hypervisor could not
// be deployed ahead of one.
func fleetDeps(cfg Config) fleetops.Deps {
	d := fleetops.Deps{
		Runner:   fleetops.NewExecRunner(),
		Node:     cfg.NodeName,
		Reboot:   fleetops.RebootHost,
		JoinPass: fleetops.JoinPasswordFromFile(cfg.JoinPasswordPath),
	}
	client, err := fleetops.NewLocalClient(cfg.ProxmoxURL, cfg.ProxmoxTokenID, cfg.ProxmoxSecret)
	if err != nil {
		// KEPT, not only logged. A refused task has to say what actually went
		// wrong, and a startup log on a hypervisor is not where anybody looks when
		// a task fails in the control plane hours later.
		slog.Warn("no Proxmox API client; fleet tasks will refuse rather than run", "error", err)
		d.ClusterErr = err
		return d
	}
	cl, err := fleetops.NewCluster(client)
	if err != nil {
		slog.Warn("could not build a cluster view; fleet tasks will refuse rather than run", "error", err)
		d.ClusterErr = err
		return d
	}
	d.Cluster = cl
	return d
}

func init() {
	startCmd.Flags().Duration("heartbeat-wait", 25*time.Second, "How long each heartbeat may wait for work")
	rootCmd.AddCommand(startCmd)
}
