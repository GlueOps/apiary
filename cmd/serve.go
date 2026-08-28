package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/glueops/apiary/internal/agentdist"
	"github.com/glueops/apiary/internal/buildinfo"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/glueops/apiary/internal/api"
	"github.com/glueops/apiary/internal/app"
	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/database"
	"github.com/glueops/apiary/internal/jobs"
	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/service"
	"github.com/glueops/apiary/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServer()
	},
}

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	httpAddr := fmt.Sprintf("%s:%s", cfg.BindHost, cfg.BindPort)

	// Said at startup, because the failure it prevents is silent and expensive.
	//
	// The embedded agent binaries are gitignored and built by one recipe nothing
	// depends on, so editing agent code and running this server ships the
	// previous build with nothing to indicate it. Diagnosing that costs the whole
	// investigation before anyone thinks to check a build artefact.
	agentdist.WarnIfStale(buildinfo.Version, func(msg string, args ...any) {
		slog.Warn(msg, args...)
	})

	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to pgx pool for River: %v", err)
	}
	defer dbPool.Close()

	controlDB, err := database.OpenControlDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open control db: %w", err)
	}

	if err := database.AssertNoDatabaseAdminRights(context.Background(), controlDB); err != nil {
		if config.IsDevEnv(cfg.Env) {
			log.Printf("WARNING: %v", err)
		} else {
			return err
		}
	}

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

		defer func() {
			if tenants != nil {
				_ = tenants.Close()
			}
		}()
	}

	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{})
	if err != nil {
		return fmt.Errorf("init river client: %w", err)
	}

	jobEnqueuer, err := jobs.NewEnqueuer(dbPool)
	if err != nil {
		return fmt.Errorf("init job enqueuer: %w", err)
	}

	tokenSvc, err := service.NewTokenService(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		time.Duration(cfg.JWTAccessTTLMin)*time.Minute,
		time.Duration(cfg.JWTRefreshTTLHour)*time.Hour,
	)
	if err != nil {
		return fmt.Errorf("init token service: %w", err)
	}

	sessionRepo := repo.NewTokenSessionRepoGorm(controlDB)
	auditRepo := repo.NewAuthAuditRepoGorm(controlDB)
	userRepo := repo.NewUserRepoGorm(controlDB)
	tokenSessions := service.NewTokenSessionService(tokenSvc, sessionRepo, auditRepo, &userRepo)

	accessRuleRepo := repo.NewAuthAccessRuleRepoGorm(controlDB)
	tokenSessions.SetAccessRules(accessRuleRepo)

	// The public/disposable mail corpus lives in auth_access_rules, not in a Go
	// map, so adding a provider is an INSERT rather than a deploy. Seeding is
	// ON CONFLICT DO NOTHING, so an operator who flips a seeded row's policy to
	// 'allow' keeps that decision across every restart.
	// AUTH_SEED_PUBLIC_DOMAIN_RULES was declared, defaulted, bound and
	// validated, but nothing read it — so the documented escape hatch for a
	// deployment whose staff legitimately use public mailboxes did nothing,
	// and DELETEing a seeded row does not stick because the seed re-inserts
	// it on the next boot.
	if !cfg.AuthSeedPublicDomainRules {
		log.Printf("AUTH_SEED_PUBLIC_DOMAIN_RULES=false: not seeding public mailbox create_org deny rules")
	} else if inserted, err := repo.SeedPublicEmailDomainRules(context.Background(), controlDB); err != nil {
		return fmt.Errorf("seed public email domain rules: %w", err)
	} else if inserted > 0 {
		log.Printf("seeded %d public email domain create_org deny rules", inserted)
	}

	// ONE gate, constructed once, shared by registration and social bootstrap.
	// Building it per-handler is how the two paths drifted into enforcing
	// different quotas in the first place.
	admissionSvc := service.NewAdmissionService(accessRuleRepo, service.OrgBootstrapLimits{
		MaxTotalOrgs:             cfg.OrgsMaxTotal,
		MaxNewOrgsPerHour:        cfg.OrgsMaxNewPerHour,
		MaxUnprovenOrgsPerDomain: cfg.AdmissionUnprovenOrgsPerDomain,
	})

	platformPolicyRepo := repo.NewPlatformPolicyRepoGorm(controlDB)
	orgPolicyRepo := repo.NewOrgPolicyRepoGorm(controlDB)
	passkeyRepo := repo.NewUserPasskeyRepoGorm(controlDB)
	policySvc := service.NewPolicyService(platformPolicyRepo, orgPolicyRepo, &userRepo, passkeyRepo)

	// The estate service needs the master key to derive the per-tenant data key
	// that seals credential material. It is constructed only when tenants are
	// available: without a tenant manager there is no tenant database to reach,
	// and a service that cannot reach one must not pretend it can.
	var estateSvc *service.EstateService
	if tenants != nil {
		masterKey, err := utils.DecodeB64(cfg.EncryptionMasterKey)
		if err != nil || len(masterKey) != utils.MasterKeyLen {
			return fmt.Errorf("estate service: ENCRYPTION_MASTER_KEY must decode to %d bytes", utils.MasterKeyLen)
		}
		estateSvc, err = service.NewEstateService(tenants, masterKey, nil)
		if err != nil {
			return fmt.Errorf("init estate service: %w", err)
		}
	} else {
		log.Printf("WARNING: no tenant manager (ENCRYPTION_MASTER_KEY unset?); estate and credential endpoints will report storage unavailable")
	}

	// Mail goes on the queue, so a slow or briefly-broken SMTP server cannot slow
	// or fail an API request, and a failed delivery is retried rather than lost.
	// The body is sealed before it is enqueued because it carries live tokens —
	// see workflow.SendEmailArgs.
	mailKey, err := utils.MasterKeyFromBase64(cfg.EncryptionMasterKey)
	if err != nil {
		return fmt.Errorf("serve: encryption master key: %w", err)
	}
	directMail := service.NewDirectEmailSender(service.NewEmailService(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom,
	))
	// The direct sender is the fallback for an enqueue failure only: losing a
	// confirmation because the queue is down is worse than sending it inline.
	mail, err := service.NewQueuedEmailSender(jobEnqueuer, mailKey, directMail)
	if err != nil {
		return fmt.Errorf("serve: mail sender: %w", err)
	}

	deps := &app.Deps{
		Config:        cfg,
		ControlDB:     controlDB,
		DBPool:        dbPool,
		River:         riverClient,
		Tenants:       tenants,
		Jobs:          jobEnqueuer,
		TokenSessions: tokenSessions,
		Policies:      policySvc,
		APIKeys:       service.NewAPIKeyService(repo.NewAPIKeyRepoGorm(controlDB)),
		Admission:     admissionSvc,
		Estates:       estateSvc,
		Mail:          mail,
	}

	apiServer, err := api.Build(*cfg, deps)
	if err != nil {
		return fmt.Errorf("api build: %w", err)
	}

	srv := &http.Server{
		Addr:           httpAddr,
		Handler:        apiServer.Router,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		urls := apiServer.PublicURLs()
		log.Printf("listening on %s (frontend_mode=%s) (vite_dev_url=%s)", httpAddr, cfg.FrontendMode, cfg.ViteDevURL)
		log.Printf("web: %s", urls["web"])
		log.Printf("docs: %s", urls["docs"])
		log.Printf("openapi: %s", urls["openapi"])

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		log.Printf("signal received: %s", sig)
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("shutting down...")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-errCh; err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
