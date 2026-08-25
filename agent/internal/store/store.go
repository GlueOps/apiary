// Package store is the bee's local source of truth.
//
// Everything the bee knows lives here, written BEFORE the action it describes.
// That ordering is the whole point: a hypervisor running "apt upgrade and
// reboot" disappears mid-task, and on the way back up the bee has to know what
// it was doing. A process that held that in memory would come back believing it
// had nothing to do, and the control plane would wait out the lease and reissue
// work that had already half happened.
//
// modernc.org/sqlite rather than mattn/go-sqlite3: no cgo, so the agent is a
// single static binary that can be dropped onto a hypervisor without a toolchain
// or matching libc.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// schema is applied on every open. Each statement is idempotent, so opening an
// existing database is the same code path as creating one — there is no
// migration step to forget on an agent that has been in the field for months.
const schema = `
CREATE TABLE IF NOT EXISTS identity (
	id             INTEGER PRIMARY KEY CHECK (id = 1),
	bee_id         TEXT NOT NULL,
	-- The ORGANIZATION, not a hive. A host enrols into an organization and joins
	-- a fleet afterwards as a task, so that a new host and one migrating between
	-- fleets follow the same path from the same starting state. There is
	-- deliberately no hive column: fleet membership is not something this host
	-- knows locally, it is what the control plane last confirmed.
	org_id         TEXT NOT NULL,
	-- The agent credential is a key/secret PAIR presented on X-AGENT-Key and
	-- X-AGENT-Secret, not a bearer token. Both halves are needed on every
	-- request, so both are stored.
	key_public     TEXT NOT NULL,
	key_secret     TEXT NOT NULL,
	base_url       TEXT NOT NULL,
	enrolled_at    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS task (
	id             TEXT PRIMARY KEY,
	kind           TEXT NOT NULL,
	spec           TEXT NOT NULL DEFAULT '{}',
	status         TEXT NOT NULL,
	phase          TEXT NOT NULL DEFAULT '',
	error          TEXT NOT NULL DEFAULT '',
	attempt        INTEGER NOT NULL DEFAULT 0,
	received_at    TIMESTAMP NOT NULL,
	started_at     TIMESTAMP,
	finished_at    TIMESTAMP,
	-- Whether the control plane has ACKNOWLEDGED the terminal outcome. A task
	-- can be finished locally and unreported: that is exactly the state a bee is
	-- in when it completes work and dies before the report lands, and it is why
	-- terminal reports have to be idempotent on the server.
	reported       INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS log_outbox (
	seq            INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id        TEXT NOT NULL,
	stream         TEXT NOT NULL,
	message        TEXT NOT NULL,
	logged_at      TIMESTAMP NOT NULL,
	shipped        INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_log_outbox_unshipped ON log_outbox (shipped, seq);
`

type Store struct{ db *sql.DB }

// Open prepares the local database.
//
// WAL and a busy timeout are not tuning: the two loops write concurrently — one
// shipping logs, one appending them — and the default journal mode would have
// them returning SQLITE_BUSY to each other during exactly the long task this
// design exists to survive.
func Open(ctx context.Context, path string) (*Store, error) {
	// THE AGENT OWNS ITS STATE DIRECTORY. Creating it here rather than expecting
	// the installer to, because the two install paths do not agree: the
	// onboarding job creates /var/lib/apiary-agent before it uploads anything,
	// while a hand install — copy the binary, run `apiary-agent enrol` — creates
	// nothing, and that is the path the packaging README documents.
	//
	// Without this the failure is SQLITE_CANTOPEN surfacing from the first query
	// as "store: apply schema: unable to open database file (14)", which names
	// neither the path nor the directory. sql.Open is lazy and cannot report it,
	// so the error arrives far from its cause.
	//
	// 0700: this database holds the agent's enrolment credential. The directory
	// is created for root and nobody else needs to traverse it.
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("store: create the state directory %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	// One writer. SQLite serialises writes anyway, and letting the pool open
	// several connections only converts that into lock contention.
	db.SetMaxOpenConns(1)

	if _, err := db.ExecContext(ctx, schema); err != nil {
		_ = db.Close()
		// The path is named because this is where an unopenable database first
		// reports itself: sql.Open only records the DSN, so the bare driver error
		// is "unable to open database file (14)" with nothing to act on.
		return nil, fmt.Errorf("store: apply schema to %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

type Identity struct {
	BeeID string
	OrgID string

	// KeyPublic identifies the credential; KeySecret authenticates it. Returned
	// once at enrolment and never reissued — a host that loses either enrols
	// again, because an endpoint that handed one back would let anyone holding a
	// bee id take the host over.
	KeyPublic string
	KeySecret string

	BaseURL    string
	EnrolledAt time.Time
}

var ErrNotEnrolled = errors.New("store: this host has not enrolled yet")

func (s *Store) SaveIdentity(ctx context.Context, id Identity) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO identity (id, bee_id, org_id, key_public, key_secret, base_url, enrolled_at)
		VALUES (1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			bee_id=excluded.bee_id, org_id=excluded.org_id,
			key_public=excluded.key_public, key_secret=excluded.key_secret,
			base_url=excluded.base_url, enrolled_at=excluded.enrolled_at`,
		id.BeeID, id.OrgID, id.KeyPublic, id.KeySecret, id.BaseURL, id.EnrolledAt)
	return err
}

func (s *Store) Identity(ctx context.Context) (*Identity, error) {
	var id Identity
	err := s.db.QueryRowContext(ctx,
		`SELECT bee_id, org_id, key_public, key_secret, base_url, enrolled_at FROM identity WHERE id = 1`).
		Scan(&id.BeeID, &id.OrgID, &id.KeyPublic, &id.KeySecret, &id.BaseURL, &id.EnrolledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotEnrolled
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// Local task states. They deliberately do not mirror the server's: the bee needs
// "finished but not yet acknowledged", which has no meaning on the other side.
const (
	TaskAccepted = "accepted"
	TaskRunning  = "running"
	TaskDone     = "done"
	TaskFailed   = "failed"
)

type Task struct {
	ID         string
	Kind       string
	Spec       string
	Status     string
	Phase      string
	Error      string
	Attempt    int
	ReceivedAt time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	Reported   bool
}

// AcceptTask records a task handed over by the control plane.
//
// Idempotent on the id, and it does NOT reset an existing row. The control plane
// re-delivers the same task to a bee that already holds it — that is the crash
// recovery path — so accepting it a second time must not roll a running task
// back to accepted, or a bee restarted mid-upgrade would begin again from the
// top having already done half the work.
func (s *Store) AcceptTask(ctx context.Context, t Task) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task (id, kind, spec, status, attempt, received_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET attempt = excluded.attempt`,
		t.ID, t.Kind, t.Spec, TaskAccepted, t.Attempt, time.Now().UTC())
	return err
}

// CurrentTask returns the one task this bee holds, if any.
func (s *Store) CurrentTask(ctx context.Context) (*Task, error) {
	var t Task
	err := s.db.QueryRowContext(ctx, `
		SELECT id, kind, spec, status, phase, error, attempt, received_at, started_at, finished_at, reported
		FROM task ORDER BY received_at DESC LIMIT 1`).
		Scan(&t.ID, &t.Kind, &t.Spec, &t.Status, &t.Phase, &t.Error, &t.Attempt,
			&t.ReceivedAt, &t.StartedAt, &t.FinishedAt, &t.Reported)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkRunning starts a task, PRESERVING any phase it had already reached.
//
// That preservation is what makes a task resumable across the reboot it caused.
// A dist-upgrade records phase="rebooting", reboots the hypervisor, and the
// agent starts again and re-enters the same task — because the control plane
// re-delivers it and the local row still says running. If starting reset the
// phase, the executor would have no way to tell "I am beginning" from "I have
// already upgraded and rebooted", and would upgrade and reboot again. In a
// rotation that is an infinite loop across a datacenter.
func (s *Store) MarkRunning(ctx context.Context, id, phase string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		UPDATE task
		SET status = ?,
		    phase = CASE WHEN phase IS NULL OR phase = '' THEN ? ELSE phase END,
		    started_at = COALESCE(started_at, ?)
		WHERE id = ?`, TaskRunning, phase, now, id)
	return err
}

func (s *Store) SetPhase(ctx context.Context, id, phase string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE task SET phase = ? WHERE id = ?`, phase, id)
	return err
}

// Finish records a terminal outcome LOCALLY. Reporting it is a separate step, so
// that a bee which finishes and immediately dies still has the outcome to send
// when it comes back.
func (s *Store) Finish(ctx context.Context, id, status, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE task SET status = ?, error = ?, finished_at = ? WHERE id = ?`,
		status, errMsg, time.Now().UTC(), id)
	return err
}

func (s *Store) MarkReported(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE task SET reported = 1 WHERE id = ?`, id)
	return err
}

// ClearTask drops a task once its outcome has been acknowledged AND its logs
// have all shipped. Dropping it earlier would discard output the control plane
// never received, which for a failed rotation is the only evidence of why.
func (s *Store) ClearTask(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM task WHERE id = ?
		AND NOT EXISTS (SELECT 1 FROM log_outbox WHERE task_id = ? AND shipped = 0)`, id, id)
	return err
}

type LogLine struct {
	Seq      int64
	TaskID   string
	Stream   string
	Message  string
	LoggedAt time.Time
}

// AppendLog writes one line to the outbox. The AUTOINCREMENT rowid IS the
// sequence the control plane deduplicates on, which means the ordering is
// established once, locally, at the moment the line is produced — not
// reconstructed later from arrival order that a retry would scramble.
//
// The counter is per BEE rather than per task, so a second task continues where
// the first left off. That is deliberate and stronger than it needs to be: the
// server's uniqueness is (task_id, seq), which a globally increasing value
// satisfies trivially, and the result is that a bee's whole output is totally
// ordered even across tasks — which is what you want when reading back a
// rotation that ran several in sequence.
func (s *Store) AppendLog(ctx context.Context, taskID, stream, message string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO log_outbox (task_id, stream, message, logged_at) VALUES (?, ?, ?, ?)`,
		taskID, stream, message, time.Now().UTC())
	return err
}

func (s *Store) UnshippedLogs(ctx context.Context, limit int) ([]LogLine, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT seq, task_id, stream, message, logged_at
		FROM log_outbox WHERE shipped = 0 ORDER BY seq ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []LogLine
	for rows.Next() {
		var l LogLine
		if err := rows.Scan(&l.Seq, &l.TaskID, &l.Stream, &l.Message, &l.LoggedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// MarkShipped is called only after the control plane has ACCEPTED the batch.
// Marking optimistically would lose the tail of a failed rotation on any network
// blip, which is precisely when the log matters most.
func (s *Store) MarkShipped(ctx context.Context, upToSeq int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE log_outbox SET shipped = 1 WHERE seq <= ? AND shipped = 0`, upToSeq)
	return err
}

// PruneShipped keeps the outbox from growing without bound on a long-lived host.
func (s *Store) PruneShipped(ctx context.Context, keep int) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM log_outbox WHERE shipped = 1 AND seq NOT IN (
			SELECT seq FROM log_outbox WHERE shipped = 1 ORDER BY seq DESC LIMIT ?)`, keep)
	return err
}
