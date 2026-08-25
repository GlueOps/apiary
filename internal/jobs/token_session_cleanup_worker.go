package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/glueops/apiary/internal/repo"
	"github.com/glueops/apiary/internal/workflow"
	"github.com/riverqueue/river"
)

const authFailureRetention = 24 * time.Hour

type TokenSessionCleanupWorker struct {
	river.WorkerDefaults[workflow.TokenSessionCleanupArgs]
	sessions repo.TokenSessionRepo
	failures repo.AuthFailureRepo
}

func NewTokenSessionCleanupWorker(sessions repo.TokenSessionRepo, failures repo.AuthFailureRepo) *TokenSessionCleanupWorker {
	if sessions == nil {
		panic("jobs: TokenSessionCleanupWorker requires a TokenSessionRepo")
	}
	if failures == nil {
		panic("jobs: TokenSessionCleanupWorker requires an AuthFailureRepo")
	}
	return &TokenSessionCleanupWorker{sessions: sessions, failures: failures}
}

func (w *TokenSessionCleanupWorker) Work(
	ctx context.Context,
	_ *river.Job[workflow.TokenSessionCleanupArgs],
) error {
	now := time.Now().UTC()

	deleted, err := w.sessions.DeleteExpired(ctx, now)
	if err != nil {
		return fmt.Errorf("delete expired token sessions: %w", err)
	}
	if deleted > 0 {
		log.Printf("[auth.token_session_cleanup] pruned %d expired sessions", deleted)
	}

	stale, err := w.failures.DeleteStale(ctx, now.Add(-authFailureRetention))
	if err != nil {
		return fmt.Errorf("delete stale auth failures: %w", err)
	}
	if stale > 0 {
		log.Printf("[auth.token_session_cleanup] pruned %d stale auth failure counters", stale)
	}
	return nil
}
