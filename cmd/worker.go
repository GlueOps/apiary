package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/glueops/apiary/internal/agentdist"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/jobs"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/utils"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start River workers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWorker()
	},
}

func runWorker() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.ValidateWorker(); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open pgx pool: %w", err)
	}
	defer pool.Close()

	controlDB, err := database.OpenControlDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open control db: %w", err)
	}

	masterKey, err := utils.MasterKeyFromBase64(cfg.EncryptionMasterKey)
	if err != nil {
		return fmt.Errorf("invalid encryption master key: %w", err)
	}

	// The worker needs tenant databases and the per-tenant key: onboarding unseals
	// the fleet key's private half, which is the thing no handler may do.
	//
	// Absent when no master key is configured, and the onboarding worker is then
	// simply not registered — see below.
	var tenants *database.TenantManager
	if strings.TrimSpace(cfg.EncryptionMasterKey) != "" {
		tenants, err = database.NewTenantManager(controlDB, cfg.EncryptionMasterKey, database.TenantPoolConfig{
			MaxSize:         cfg.TenantPoolMaxSize,
			MaxConns:        cfg.TenantPoolMaxConns,
			MaxIdleConns:    cfg.TenantPoolMaxIdleConns,
			ConnMaxIdleTime: time.Duration(cfg.TenantPoolConnMaxIdleMin) * time.Minute,
			ConnMaxLifetime: time.Duration(cfg.TenantPoolConnMaxLifetimeMin) * time.Minute,
			EvictGrace:      time.Duration(cfg.TenantPoolEvictGraceSec) * time.Second,
		})
		if err != nil {
			return fmt.Errorf("init tenant manager: %w", err)
		}
	}

	workers := river.NewWorkers()

	// Declared before the workers because one of them enqueues jobs through it,
	// and assigned after the client exists. The fan-out worker cannot run before
	// that assignment — nothing is processing the queue until Start — so the nil
	// check inside it is a guard against a wiring mistake, not a race.
	var riverClient *river.Client[pgx.Tx]

	provisioner, err := jobs.NewTenantProvisionerService(controlDB, cfg.DatabaseURL, cfg.ProvisionerDatabaseURL, masterKey)
	if err != nil {
		return fmt.Errorf("init tenant provisioner: %w", err)
	}
	destroyer := jobs.NewTenantDestroyerService(controlDB, cfg.DatabaseURL, cfg.ProvisionerDatabaseURL)

	river.AddWorker(workers, jobs.NewProvisionTenantWorker(provisioner))
	river.AddWorker(workers, jobs.NewDestroyTenantWorker(destroyer))

	sessionRepo := repo.NewTokenSessionRepoGorm(controlDB)

	river.AddWorker(workers, jobs.NewTokenSessionCleanupWorker(sessionRepo, repo.NewAuthFailureRepoGorm(controlDB)))

	river.AddWorker(workers, jobs.NewOrgProvingWorker(controlDB, cfg.AdmissionProvenDistinctDays))

	// Hypervisor onboarding.
	//
	// Only wired when a master key is configured, because it must unseal the
	// fleet key's private half. A worker registered without one would accept jobs
	// and fail every attempt — a queue that looks busy and achieves nothing is
	// worse than a job kind that is plainly absent.
	if tenants != nil {
		sshKeys := service.NewSSHKeyService(tenants)
		firewalls := service.NewFirewallService(tenants)
		fleetRepoFor := func(ctx context.Context, orgID uuid.UUID) (*gorm.DB, error) {
			tCtx, err := tenants.GetTenantContext(ctx, orgID.String())
			if err != nil {
				return nil, err
			}
			return tCtx.DB, nil
		}

		// Membership changes ask for a fresh observation instead of waiting for the
		// tick, so placement never spends a sync interval deciding against a fleet
		// that no longer exists. Shared by departure and arrival because it is the
		// same request about a different hive.
		//
		// riverClient is read at CALL time rather than captured: it is assigned
		// below, after AddWorker, and no job can have run before then because
		// nothing is processing the queue until Start. The nil check therefore
		// guards a wiring mistake, not a race — the same deferred-closure shape the
		// capacity sync fan-out already uses.
		// The same insert, scheduled rather than immediate.
		//
		// Arrival uses this because Proxmox has not populated /cluster/resources
		// stats for a node at the instant it joins: an immediate sync records zero
		// cores and zero memory and stamps last_synced_at, so the figures read as
		// observed rather than missing.
		//
		// It must be the ONLY sync arrival enqueues. SyncHiveCapacityArgs is unique
		// ByArgs across five states including scheduled, and River reports a unique
		// collision as a successful insert carrying UniqueSkippedAsDuplicate rather
		// than an error — so pairing this with an immediate enqueue would drop one
		// of them silently, with nothing to see in a log.
		enqueueCapacitySyncAfter := func(ctx context.Context, orgID, hiveID uuid.UUID,
			after time.Duration) error {
			if riverClient == nil {
				return fmt.Errorf("capacity sync trigger: the river client is not ready")
			}
			_, err := riverClient.Insert(ctx, jobs.SyncHiveCapacityArgs{
				OrganizationID: orgID.String(),
				HiveID:         hiveID.String(),
			}, &river.InsertOpts{ScheduledAt: time.Now().Add(after)})
			return err
		}

		enqueueCapacitySync := func(ctx context.Context, orgID, hiveID uuid.UUID) error {
			if riverClient == nil {
				return fmt.Errorf("capacity sync trigger: the river client is not ready")
			}
			// jobs.SyncHiveCapacityArgs, NOT workflow.SyncHiveCapacityArgs. Only this
			// one declares InsertOpts with ByArgs uniqueness across the five live
			// states, and that is the whole reason an extra enqueue is safe: a
			// membership-triggered sync coalesces with whatever the schedule already
			// has in flight for the hive rather than stacking behind it. River reports
			// a unique collision as a successful insert carrying
			// UniqueSkippedAsDuplicate, not as an error, so there is nothing to
			// special-case here.
			_, err := riverClient.Insert(ctx, jobs.SyncHiveCapacityArgs{
				OrganizationID: orgID.String(),
				HiveID:         hiveID.String(),
			}, nil)
			return err
		}

		// One inventory service, constructed before the workers that use it. It
		// resolves a fleet's Proxmox credential, which onboarding needs to hand the
		// agent a token and the capacity sync needs to observe the cluster — the
		// same credential, read the same way, so the two cannot disagree about
		// which token a fleet uses.
		inventory := service.NewProxmoxInventoryService(tenants, repo.DefaultProviderScopes)

		river.AddWorker(workers, jobs.NewOnboardHypervisorWorker(jobs.OnboardDeps{
			FleetKey: func(ctx context.Context, orgID uuid.UUID) (string, []byte, error) {
				k, err := sshKeys.ActiveFleetKey(ctx, orgID)
				if err != nil {
					return "", nil, err
				}
				priv, err := sshKeys.PrivateKey(ctx, orgID, k.ID)
				if err != nil {
					return "", nil, err
				}
				return k.PublicKey, priv, nil
			},
			ManagementSets: firewalls.ManagementSets,
			// The token the host redeems for its own credential.
			//
			// Unwired until now, which is what made every automated onboarding
			// fail at the enrol step with "no enrolment token source configured" —
			// the field was declared and nil-checked and never assigned.
			//
			// Single-use and ten minutes, because this one is not an operator's
			// batch token for bringing up a rack: it is minted for ONE host and
			// redeemed within seconds of arriving. CreatedByID is nil — a worker is
			// not a person, and attributing the mint to one would put a name
			// against an action nobody took.
			EnrolmentToken: func(ctx context.Context, orgID uuid.UUID) (string, error) {
				raw, hash, err := service.NewEnrolmentToken()
				if err != nil {
					return "", fmt.Errorf("mint an enrolment token: %w", err)
				}
				row := control.BeeEnrolmentToken{
					OrganizationID: orgID,
					TokenHash:      hash,
					MaxUses:        1,
					ExpiresAt:      time.Now().UTC().Add(10 * time.Minute),
				}
				if err := controlDB.WithContext(ctx).Create(&row).Error; err != nil {
					return "", fmt.Errorf("record the enrolment token: %w", err)
				}
				return raw, nil
			},
			NewRunner: func(addr, user string, priv []byte) (jobs.Runner, error) {
				return jobs.NewSSHRunner(addr, user, priv)
			},
			Heartbeated: func(ctx context.Context, orgID uuid.UUID, machineID string, since time.Time) (bool, error) {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return false, err
				}
				return repo.Heartbeated(ctx, db, machineID, since)
			},
			// One source for the agent binary, shared with the self-upgrade endpoint.
			// Onboarding and upgrading install the same bytes, so a host brought in
			// today runs what an upgraded host runs.
			Bundle: func(goarch string) (jobs.AgentBundle, error) {
				bin, err := agentdist.Binary("linux", goarch)
				if err != nil {
					return jobs.AgentBundle{}, fmt.Errorf("onboard: this control plane carries "+
						"no agent binary for linux/%s: %w", goarch, err)
				}
				return jobs.AgentBundle{Binary: bin}, nil
			},
			BaseURL: cfg.BaseURL,
			// The machine's fleet credential. Resolved at onboarding time rather
			// than captured, because which fleet a machine belongs to is a fact
			// about the row and changes when it migrates.
			ProxmoxCredential: func(ctx context.Context, orgID, hvID uuid.UUID) (jobs.ProxmoxCredential, error) {
				tokenID, secret, apiURL, err := inventory.AgentProxmoxCredential(ctx, orgID, hvID)
				if err != nil {
					return jobs.ProxmoxCredential{}, err
				}
				return jobs.ProxmoxCredential{TokenID: tokenID, Secret: secret, APIURL: apiURL}, nil
			},
			// This machine's own SSH host keys, captured while onboarding holds an
			// authenticated session to it. They are for OTHER machines: a host
			// joining a cluster must pin its PEER before pvecm add runs
			// ssh-copy-id, and under agent-polled convergence there is no longer a
			// moment where the control plane holds SSH to both at once.
			RecordHostKeys: func(ctx context.Context, orgID, hvID uuid.UUID, keys []string) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.RecordHostKeys(ctx, db, hvID, keys)
			},
			RecordOutcome: func(ctx context.Context, orgID, hvID uuid.UUID, onboardErr error) (bool, error) {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return false, err
				}
				return repo.RecordOnboardOutcome(ctx, db, hvID, onboardErr)
			},
		}))

		// Hypervisor departure: removing a host from its hive and leaving it
		// standalone. Shares the fleet key with onboarding, and for the same
		// reason — it needs SSH to the metal, on two hosts rather than one.
		river.AddWorker(workers, jobs.NewDepartHypervisorWorker(jobs.DepartDeps{
			RecordOutcome: func(ctx context.Context, orgID, hvID uuid.UUID,
				operation string, opErr error) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.RecordOperationOutcome(ctx, db, hvID, operation, opErr)
			},
			FleetKey: func(ctx context.Context, orgID uuid.UUID) (string, []byte, error) {
				k, err := sshKeys.ActiveFleetKey(ctx, orgID)
				if err != nil {
					return "", nil, err
				}
				priv, err := sshKeys.PrivateKey(ctx, orgID, k.ID)
				if err != nil {
					return "", nil, err
				}
				return k.PublicKey, priv, nil
			},
			Snapshot: func(ctx context.Context, orgID, hypID uuid.UUID) (jobs.DepartureSnapshot, error) {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return jobs.DepartureSnapshot{}, err
				}
				return jobs.DepartureSnapshotFromDB(ctx, db, hypID)
			},
			NewRunner: func(addr, user string, priv []byte) (jobs.Runner, error) {
				return jobs.NewSSHRunner(addr, user, priv)
			},
			MarkFleetless: func(ctx context.Context, orgID, hypervisorID uuid.UUID) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.MarkFleetless(ctx, db, hypervisorID)
			},
			EnqueueCapacitySync: enqueueCapacitySync,
			Thresholds:          tenant.DrainThresholds{ObservationMaxAge: cfg.FleetObservationMaxAge},
		}))

		// Hypervisor arrival: adding a standalone host to a hive. The mirror of
		// departure, and a separate job on purpose — a migration is a departure
		// and then an arrival, with a real fleetless state between them.
		river.AddWorker(workers, jobs.NewArriveHypervisorWorker(jobs.ArriveDeps{
			// The TARGET fleet's credential, written after the join — a join replaces
			// the node's token database with the cluster's, so whatever the agent
			// held before no longer exists on that host.
			ProxmoxCredential: func(ctx context.Context, orgID, hiveID uuid.UUID) (jobs.ProxmoxCredential, error) {
				tokenID, secret, apiURL, err := inventory.AgentProxmoxCredentialForHive(ctx, orgID, hiveID)
				if err != nil {
					return jobs.ProxmoxCredential{}, err
				}
				return jobs.ProxmoxCredential{TokenID: tokenID, Secret: secret, APIURL: apiURL}, nil
			},
			BaseURL: cfg.BaseURL,
			RecordOutcome: func(ctx context.Context, orgID, hvID uuid.UUID,
				operation string, opErr error) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.RecordOperationOutcome(ctx, db, hvID, operation, opErr)
			},
			FleetKey: func(ctx context.Context, orgID uuid.UUID) (string, []byte, error) {
				k, err := sshKeys.ActiveFleetKey(ctx, orgID)
				if err != nil {
					return "", nil, err
				}
				priv, err := sshKeys.PrivateKey(ctx, orgID, k.ID)
				if err != nil {
					return "", nil, err
				}
				return k.PublicKey, priv, nil
			},
			Snapshot: func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID) (jobs.ArrivalSnapshot, error) {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return jobs.ArrivalSnapshot{}, err
				}
				return jobs.ArrivalSnapshotFromDB(ctx, db, hypervisorID, hiveID)
			},
			NewRunner: func(addr, user string, priv []byte) (jobs.Runner, error) {
				return jobs.NewSSHRunner(addr, user, priv)
			},
			MarkJoined: func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID, nodeName string) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.MarkJoined(ctx, db, hypervisorID, hiveID, nodeName)
			},
			// The DELAYED one, and only it — see enqueueCapacitySyncAfter for why
			// pairing the two would silently drop one.
			EnqueueCapacitySyncAfter: enqueueCapacitySyncAfter,
		}))

		// Hive bootstrap: creating the FIRST cluster, on one standalone host.
		//
		// Without this registered, a hive created in the UI could never acquire a
		// node: arrival refuses with target_hive_empty until the hive has one, and
		// nothing else creates a cluster. The job existed and nothing enqueued it,
		// which is the same as it not existing.
		river.AddWorker(workers, jobs.NewBootstrapHiveWorker(jobs.BootstrapDeps{
			FleetKey: func(ctx context.Context, orgID uuid.UUID) (string, []byte, error) {
				k, err := sshKeys.ActiveFleetKey(ctx, orgID)
				if err != nil {
					return "", nil, err
				}
				priv, err := sshKeys.PrivateKey(ctx, orgID, k.ID)
				if err != nil {
					return "", nil, err
				}
				return k.PublicKey, priv, nil
			},
			Snapshot: func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID) (jobs.BootstrapSnapshot, error) {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return jobs.BootstrapSnapshot{}, err
				}
				return jobs.BootstrapSnapshotFromDB(ctx, db, hypervisorID, hiveID)
			},
			NewRunner: func(addr, user string, priv []byte) (jobs.Runner, error) {
				return jobs.NewSSHRunner(addr, user, priv)
			},
			// The SAME function the arrival uses, deliberately: a node that became
			// a cluster's first member and one that joined an existing cluster end
			// up in identical states, unschedulable included — nothing has observed
			// this cluster's capacity yet.
			MarkJoined: func(ctx context.Context, orgID, hypervisorID, hiveID uuid.UUID, nodeName string) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.MarkJoined(ctx, db, hypervisorID, hiveID, nodeName)
			},
			RecordOutcome: func(ctx context.Context, orgID, hvID uuid.UUID,
				operation string, opErr error) error {
				db, err := fleetRepoFor(ctx, orgID)
				if err != nil {
					return err
				}
				return repo.RecordOperationOutcome(ctx, db, hvID, operation, opErr)
			},
			EnqueueCapacitySyncAfter: enqueueCapacitySyncAfter,
			EnqueueCapacitySync:      enqueueCapacitySync,
		}))

		// Capacity sync. Feeds ObservedGuests, which the departure and arrival
		// gates read — without it every observation is absent, the drain state is
		// unknown, and every departure refuses. Safe, but not a working system.
		syncDeps := jobs.CapacitySyncDeps{
			Inventory:     inventory.Inventory,
			NodeAddresses: inventory.NodeAddresses,
			// Without this a standalone host is recorded as the hive's only
			// member, which closes every route out — see the field's own comment.
			ClusterIdentity: inventory.ClusterIdentity,
			TenantDB:        fleetRepoFor,
		}
		river.AddWorker(workers, jobs.NewSyncHiveCapacityWorker(syncDeps))
		river.AddWorker(workers, jobs.NewSyncAllHivesWorker(controlDB, fleetRepoFor,
			func(ctx context.Context, args river.JobArgs) error {
				// Insert through the client this worker process owns. Assigned below,
				// after the client is constructed — the fan-out job cannot run before
				// then, because nothing is processing the queue yet.
				if riverClient == nil {
					return fmt.Errorf("capacity sync: the river client is not ready")
				}
				_, err := riverClient.Insert(ctx, args, nil)
				return err
			}))

		// The onboarding tick. Registered beside the capacity sync fan-out
		// because it is the same shape — enumerate tenants, enqueue per row, do
		// no remote work here — and shares its insert closure's reason for
		// existing.
		river.AddWorker(workers, jobs.NewClaimPendingOnboardsWorker(controlDB, fleetRepoFor,
			func(ctx context.Context, args river.JobArgs) error {
				if riverClient == nil {
					return fmt.Errorf("onboarding tick: the river client is not ready")
				}
				_, err := riverClient.Insert(ctx, args, nil)
				return err
			}))
	}

	river.AddWorker(workers, jobs.NewDomainCohortAuditWorker(controlDB, jobs.CohortAuditOptions{
		MinSample:   cfg.AdmissionCohortMinSample,
		ProvenRatio: cfg.AdmissionCohortProvenRatio,
	}))

	// Outbound mail. Registered here rather than in the API process because that
	// is the whole point of queueing it: a request returns as soon as the job is
	// written, and delivery — including its retries — happens out of band.
	// Config.Validate has already checked this decodes, so a failure here means
	// the process is misconfigured in a way that would make every queued email
	// undeliverable. Refuse to start rather than retry-loop the whole queue.
	mailKey, err := utils.MasterKeyFromBase64(cfg.EncryptionMasterKey)
	if err != nil {
		return fmt.Errorf("worker: encryption master key: %w", err)
	}
	river.AddWorker(workers, jobs.NewSendEmailWorker(
		service.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom),
		mailKey,
	))

	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 50},
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(24*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return &workflow.TokenSessionCleanupArgs{}, &river.InsertOpts{}
				},
				&river.PeriodicJobOpts{
					ID: "token-session-cleanup",
				},
			),

			river.NewPeriodicJob(
				river.PeriodicInterval(time.Duration(cfg.AdmissionProvingIntervalHours)*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return &workflow.OrgProvingArgs{}, &river.InsertOpts{}
				},
				&river.PeriodicJobOpts{
					ID: "admission-org-proving",
				},
			),

			// Capacity sync fan-out. RunOnStart so a freshly started worker does
			// not leave the fleet unobserved until the first interval elapses —
			// during which every departure would refuse with drain_unknown.
			river.NewPeriodicJob(
				river.PeriodicInterval(cfg.FleetCapacitySyncInterval),
				func() (river.JobArgs, *river.InsertOpts) {
					return &jobs.SyncAllHivesArgs{}, &river.InsertOpts{}
				},
				&river.PeriodicJobOpts{
					ID:         "fleet-capacity-sync",
					RunOnStart: true,
				},
			),

			// Onboarding tick. RunOnStart because a worker restarting during an
			// onboard leaves rows in `provisioning` with leases that outlive the
			// process; waiting a full interval before looking would add that
			// interval to every machine already waiting.
			river.NewPeriodicJob(
				river.PeriodicInterval(cfg.FleetOnboardingTickInterval),
				func() (river.JobArgs, *river.InsertOpts) {
					return &jobs.ClaimPendingOnboardsArgs{}, &river.InsertOpts{}
				},
				&river.PeriodicJobOpts{
					ID:         "fleet-onboarding-tick",
					RunOnStart: true,
				},
			),

			river.NewPeriodicJob(
				river.PeriodicInterval(time.Duration(cfg.AdmissionCohortIntervalHours)*time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return &workflow.DomainCohortAuditArgs{}, &river.InsertOpts{}
				},
				&river.PeriodicJobOpts{
					ID: "admission-domain-cohort-audit",
				},
			),
		},
	})
	if err != nil {
		return fmt.Errorf("new river client: %w", err)
	}
	riverClient = client

	log.Printf("worker starting")
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start river client: %w", err)
	}

	<-ctx.Done()
	log.Printf("worker stopping")
	return client.Stop(context.Background())
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
