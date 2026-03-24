package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dvelton/gh-flair/internal/model"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

const timeLayout = time.RFC3339Nano

// Store wraps a SQLite database and exposes typed persistence operations.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at dbPath and runs any pending
// schema migrations. The directory is created automatically if needed.
func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("store: create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	// SQLite performs best with a single writer connection.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: pragma: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// ---------------------------------------------------------------------------
// Migration
// ---------------------------------------------------------------------------

func (s *Store) migrate() error {
	if _, err := s.db.Exec(createSchemaVersionTable); err != nil {
		return fmt.Errorf("store: create schema_version: %w", err)
	}

	var current int
	row := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("store: read schema version: %w", err)
	}

	for i := current; i < currentSchemaVersion; i++ {
		ddl := migrations[i]
		if _, err := s.db.Exec(ddl); err != nil {
			return fmt.Errorf("store: migration %d: %w", i+1, err)
		}
		_, err := s.db.Exec(
			`INSERT INTO schema_version(version, applied_at) VALUES (?, ?)`,
			i+1, time.Now().UTC().Format(timeLayout),
		)
		if err != nil {
			return fmt.Errorf("store: record migration %d: %w", i+1, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Repo CRUD
// ---------------------------------------------------------------------------

// AddRepo inserts a new repo and sets its generated ID on the value.
func (s *Store) AddRepo(r *model.Repo) error {
	pkgsJSON, err := json.Marshal(r.Packages)
	if err != nil {
		return fmt.Errorf("store: marshal packages: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO repos(owner, name, full_name, packages, added_at)
		 VALUES (?, ?, ?, ?, ?)`,
		r.Owner, r.Name, r.FullName, string(pkgsJSON),
		r.AddedAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("store: add repo: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

// GetRepo retrieves a single repo by its full name ("owner/name").
func (s *Store) GetRepo(fullName string) (*model.Repo, error) {
	row := s.db.QueryRow(
		`SELECT id, owner, name, full_name, packages, added_at
		 FROM repos WHERE full_name = ?`, fullName,
	)
	return scanRepo(row)
}

// ListRepos returns all tracked repositories.
func (s *Store) ListRepos() ([]model.Repo, error) {
	rows, err := s.db.Query(
		`SELECT id, owner, name, full_name, packages, added_at FROM repos ORDER BY added_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list repos: %w", err)
	}
	defer rows.Close()

	var repos []model.Repo
	for rows.Next() {
		r, err := scanRepo(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, *r)
	}
	return repos, rows.Err()
}

// RemoveRepo deletes a repo and all its associated data (cascades via FK).
func (s *Store) RemoveRepo(fullName string) error {
	_, err := s.db.Exec(`DELETE FROM repos WHERE full_name = ?`, fullName)
	return err
}

type repoScanner interface {
	Scan(dest ...any) error
}

func scanRepo(s repoScanner) (*model.Repo, error) {
	var r model.Repo
	var pkgsJSON, addedAtStr string
	if err := s.Scan(&r.ID, &r.Owner, &r.Name, &r.FullName, &pkgsJSON, &addedAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan repo: %w", err)
	}
	if err := json.Unmarshal([]byte(pkgsJSON), &r.Packages); err != nil {
		return nil, fmt.Errorf("store: unmarshal packages: %w", err)
	}
	t, err := time.Parse(timeLayout, addedAtStr)
	if err != nil {
		return nil, fmt.Errorf("store: parse added_at: %w", err)
	}
	r.AddedAt = t
	return &r, nil
}

// ---------------------------------------------------------------------------
// Snapshots
// ---------------------------------------------------------------------------

// SaveSnapshot persists a snapshot and sets its generated ID.
func (s *Store) SaveSnapshot(snap *model.Snapshot) error {
	res, err := s.db.Exec(
		`INSERT INTO snapshots(repo_id, stars, forks, open_prs, taken_at)
		 VALUES (?, ?, ?, ?, ?)`,
		snap.RepoID, snap.Stars, snap.Forks, snap.OpenPRs,
		snap.TakenAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("store: save snapshot: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	snap.ID = id
	return nil
}

// GetLatestSnapshot returns the most recent snapshot for the given repo.
func (s *Store) GetLatestSnapshot(repoID int64) (*model.Snapshot, error) {
	row := s.db.QueryRow(
		`SELECT id, repo_id, stars, forks, open_prs, taken_at
		 FROM snapshots WHERE repo_id = ?
		 ORDER BY taken_at DESC LIMIT 1`, repoID,
	)
	return scanSnapshot(row)
}

// GetSnapshotBefore returns the most recent snapshot taken before (exclusive)
// the given time for the repo. Returns nil when none exist.
func (s *Store) GetSnapshotBefore(repoID int64, before time.Time) (*model.Snapshot, error) {
	row := s.db.QueryRow(
		`SELECT id, repo_id, stars, forks, open_prs, taken_at
		 FROM snapshots WHERE repo_id = ? AND taken_at < ?
		 ORDER BY taken_at DESC LIMIT 1`,
		repoID, before.UTC().Format(timeLayout),
	)
	return scanSnapshot(row)
}

func scanSnapshot(row *sql.Row) (*model.Snapshot, error) {
	var snap model.Snapshot
	var takenAtStr string
	if err := row.Scan(&snap.ID, &snap.RepoID, &snap.Stars, &snap.Forks, &snap.OpenPRs, &takenAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan snapshot: %w", err)
	}
	t, err := time.Parse(timeLayout, takenAtStr)
	if err != nil {
		return nil, fmt.Errorf("store: parse taken_at: %w", err)
	}
	snap.TakenAt = t
	return &snap, nil
}

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

// SaveEvents inserts a batch of events inside a single transaction. Any event
// with an empty ID is assigned a new UUID.
func (s *Store) SaveEvents(events []model.Event) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(
		`INSERT OR IGNORE INTO events
		 (id, repo_id, kind, title, body, url, actor, meta, occurred_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("store: prepare event insert: %w", err)
	}
	defer stmt.Close()

	for i := range events {
		e := &events[i]
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		metaJSON, err := json.Marshal(e.Meta)
		if err != nil {
			return fmt.Errorf("store: marshal meta: %w", err)
		}
		if e.CreatedAt.IsZero() {
			e.CreatedAt = time.Now().UTC()
		}
		_, err = stmt.Exec(
			e.ID, e.RepoID, string(e.Kind),
			e.Title, e.Body, e.URL, e.Actor,
			string(metaJSON),
			e.OccuredAt.UTC().Format(timeLayout),
			e.CreatedAt.UTC().Format(timeLayout),
		)
		if err != nil {
			return fmt.Errorf("store: insert event %s: %w", e.ID, err)
		}
	}
	return tx.Commit()
}

// GetEventByID returns the event with the given ID, or nil if not found.
func (s *Store) GetEventByID(id string) (*model.Event, error) {
	row := s.db.QueryRow(
		`SELECT id, repo_id, kind, title, body, url, actor, meta, occurred_at, created_at
		 FROM events WHERE id = ?`, id,
	)
	return scanEvent(row)
}

// GetEventsSince returns events for a repo that occurred on or after `since`.
// Optionally filter to specific EventKind values; pass nil/empty to get all.
func (s *Store) GetEventsSince(repoID int64, since time.Time, kinds []model.EventKind) ([]model.Event, error) {
	query := `SELECT id, repo_id, kind, title, body, url, actor, meta, occurred_at, created_at
	          FROM events WHERE repo_id = ? AND occurred_at >= ?`
	args := []any{repoID, since.UTC().Format(timeLayout)}

	if len(kinds) > 0 {
		placeholders := make([]string, len(kinds))
		for i, k := range kinds {
			placeholders[i] = "?"
			args = append(args, string(k))
		}
		query += " AND kind IN (" + strings.Join(placeholders, ",") + ")"
	}
	query += " ORDER BY occurred_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: get events since: %w", err)
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		e, err := scanEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, rows.Err()
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row *sql.Row) (*model.Event, error) {
	var e model.Event
	var kindStr, metaJSON, occurredAtStr, createdAtStr string
	if err := row.Scan(&e.ID, &e.RepoID, &kindStr, &e.Title, &e.Body, &e.URL, &e.Actor, &metaJSON, &occurredAtStr, &createdAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan event: %w", err)
	}
	return finishEvent(&e, kindStr, metaJSON, occurredAtStr, createdAtStr)
}

func scanEventRow(rows *sql.Rows) (*model.Event, error) {
	var e model.Event
	var kindStr, metaJSON, occurredAtStr, createdAtStr string
	if err := rows.Scan(&e.ID, &e.RepoID, &kindStr, &e.Title, &e.Body, &e.URL, &e.Actor, &metaJSON, &occurredAtStr, &createdAtStr); err != nil {
		return nil, fmt.Errorf("store: scan event row: %w", err)
	}
	return finishEvent(&e, kindStr, metaJSON, occurredAtStr, createdAtStr)
}

func finishEvent(e *model.Event, kindStr, metaJSON, occurredAtStr, createdAtStr string) (*model.Event, error) {
	e.Kind = model.EventKind(kindStr)
	if err := json.Unmarshal([]byte(metaJSON), &e.Meta); err != nil {
		return nil, fmt.Errorf("store: unmarshal meta: %w", err)
	}
	oAt, err := time.Parse(timeLayout, occurredAtStr)
	if err != nil {
		return nil, fmt.Errorf("store: parse occurred_at: %w", err)
	}
	cAt, err := time.Parse(timeLayout, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("store: parse created_at: %w", err)
	}
	e.OccuredAt = oAt
	e.CreatedAt = cAt
	return e, nil
}

// ---------------------------------------------------------------------------
// Milestones
// ---------------------------------------------------------------------------

// SaveMilestone persists a milestone record (upsert on repo+kind+threshold).
func (s *Store) SaveMilestone(m *model.Milestone) error {
	res, err := s.db.Exec(
		`INSERT INTO milestones(repo_id, kind, threshold, actual_value, celebrated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(repo_id, kind, threshold) DO UPDATE SET
		   actual_value  = excluded.actual_value,
		   celebrated_at = excluded.celebrated_at`,
		m.RepoID, string(m.Kind), m.Threshold, m.ActualValue,
		m.CelebratedAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("store: save milestone: %w", err)
	}
	if m.ID == 0 {
		id, err := res.LastInsertId()
		if err == nil {
			m.ID = id
		}
	}
	return nil
}

// GetLastMilestone returns the highest-threshold milestone celebrated for the
// given repo and kind, or nil if none exist.
func (s *Store) GetLastMilestone(repoID int64, kind model.MilestoneKind) (*model.Milestone, error) {
	row := s.db.QueryRow(
		`SELECT id, repo_id, kind, threshold, actual_value, celebrated_at
		 FROM milestones WHERE repo_id = ? AND kind = ?
		 ORDER BY threshold DESC LIMIT 1`,
		repoID, string(kind),
	)
	return scanMilestone(row)
}

// IsMilestoneCelebrated reports whether the given threshold has already been
// recorded for the repo+kind combination.
func (s *Store) IsMilestoneCelebrated(repoID int64, kind model.MilestoneKind, threshold int) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM milestones WHERE repo_id = ? AND kind = ? AND threshold = ?`,
		repoID, string(kind), threshold,
	).Scan(&count)
	return count > 0, err
}

func scanMilestone(row *sql.Row) (*model.Milestone, error) {
	var m model.Milestone
	var kindStr, celebratedAtStr string
	if err := row.Scan(&m.ID, &m.RepoID, &kindStr, &m.Threshold, &m.ActualValue, &celebratedAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan milestone: %w", err)
	}
	m.Kind = model.MilestoneKind(kindStr)
	t, err := time.Parse(timeLayout, celebratedAtStr)
	if err != nil {
		return nil, fmt.Errorf("store: parse celebrated_at: %w", err)
	}
	m.CelebratedAt = t
	return &m, nil
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

// SaveSession records the current run timestamp and sets the generated ID.
func (s *Store) SaveSession(sess *model.Session) error {
	res, err := s.db.Exec(
		`INSERT INTO sessions(ran_at) VALUES (?)`,
		sess.RanAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("store: save session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	sess.ID = id
	return nil
}

// GetLastSession returns the most recently recorded session, or nil if none.
func (s *Store) GetLastSession() (*model.Session, error) {
	row := s.db.QueryRow(
		`SELECT id, ran_at FROM sessions ORDER BY ran_at DESC LIMIT 1`,
	)
	var sess model.Session
	var ranAtStr string
	if err := row.Scan(&sess.ID, &ranAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: scan session: %w", err)
	}
	t, err := time.Parse(timeLayout, ranAtStr)
	if err != nil {
		return nil, fmt.Errorf("store: parse ran_at: %w", err)
	}
	sess.RanAt = t
	return &sess, nil
}

// ---------------------------------------------------------------------------
// Streaks
// ---------------------------------------------------------------------------

// SaveStreak upserts the streak record for a repo+metric pair.
func (s *Store) SaveStreak(streak *model.Streak) error {
	res, err := s.db.Exec(
		`INSERT INTO streaks(repo_id, metric, current_days, best_days, last_active)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(repo_id, metric) DO UPDATE SET
		   current_days = excluded.current_days,
		   best_days    = excluded.best_days,
		   last_active  = excluded.last_active`,
		streak.RepoID, streak.Metric, streak.CurrentDays, streak.BestDays,
		streak.LastActive.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("store: save streak: %w", err)
	}
	if streak.ID == 0 {
		id, err := res.LastInsertId()
		if err == nil {
			streak.ID = id
		}
	}
	return nil
}

// GetStreaks returns all streak records for the given repo.
func (s *Store) GetStreaks(repoID int64) ([]model.Streak, error) {
	rows, err := s.db.Query(
		`SELECT id, repo_id, metric, current_days, best_days, last_active
		 FROM streaks WHERE repo_id = ? ORDER BY metric`,
		repoID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get streaks: %w", err)
	}
	defer rows.Close()
	return scanStreaks(rows)
}

// GetAllStreaks returns every streak row across all repos.
func (s *Store) GetAllStreaks() ([]model.Streak, error) {
	rows, err := s.db.Query(
		`SELECT id, repo_id, metric, current_days, best_days, last_active
		 FROM streaks ORDER BY repo_id, metric`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get all streaks: %w", err)
	}
	defer rows.Close()
	return scanStreaks(rows)
}

func scanStreaks(rows *sql.Rows) ([]model.Streak, error) {
	var streaks []model.Streak
	for rows.Next() {
		var st model.Streak
		var lastActiveStr string
		if err := rows.Scan(&st.ID, &st.RepoID, &st.Metric, &st.CurrentDays, &st.BestDays, &lastActiveStr); err != nil {
			return nil, fmt.Errorf("store: scan streak: %w", err)
		}
		t, err := time.Parse(timeLayout, lastActiveStr)
		if err != nil {
			return nil, fmt.Errorf("store: parse last_active: %w", err)
		}
		st.LastActive = t
		streaks = append(streaks, st)
	}
	return streaks, rows.Err()
}

// ---------------------------------------------------------------------------
// SavedMoments
// ---------------------------------------------------------------------------

// MomentFilters controls which saved moments are returned by ListMoments.
type MomentFilters struct {
	RepoFullName string    // exact match on repos.full_name; empty = all repos
	Since        time.Time // zero = no lower bound
	Until        time.Time // zero = no upper bound
}

// SaveMoment persists a saved moment and sets its generated ID.
func (s *Store) SaveMoment(m *model.SavedMoment) error {
	res, err := s.db.Exec(
		`INSERT INTO saved_moments(event_id, repo_id, kind, title, body, actor, url, saved_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		m.EventID, m.RepoID, string(m.Kind),
		m.Title, m.Body, m.Actor, m.URL,
		m.SavedAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("store: save moment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

// ListMoments returns saved moments, optionally filtered.
func (s *Store) ListMoments(f MomentFilters) ([]model.SavedMoment, error) {
	query := `SELECT sm.id, sm.event_id, sm.repo_id, sm.kind,
	                 sm.title, sm.body, sm.actor, sm.url, sm.saved_at
	          FROM saved_moments sm`
	var conditions []string
	var args []any

	if f.RepoFullName != "" {
		query += ` JOIN repos r ON r.id = sm.repo_id`
		conditions = append(conditions, "r.full_name = ?")
		args = append(args, f.RepoFullName)
	}
	if !f.Since.IsZero() {
		conditions = append(conditions, "sm.saved_at >= ?")
		args = append(args, f.Since.UTC().Format(timeLayout))
	}
	if !f.Until.IsZero() {
		conditions = append(conditions, "sm.saved_at <= ?")
		args = append(args, f.Until.UTC().Format(timeLayout))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY sm.saved_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list moments: %w", err)
	}
	defer rows.Close()

	var moments []model.SavedMoment
	for rows.Next() {
		var m model.SavedMoment
		var kindStr, savedAtStr string
		if err := rows.Scan(&m.ID, &m.EventID, &m.RepoID, &kindStr,
			&m.Title, &m.Body, &m.Actor, &m.URL, &savedAtStr); err != nil {
			return nil, fmt.Errorf("store: scan moment: %w", err)
		}
		m.Kind = model.EventKind(kindStr)
		t, err := time.Parse(timeLayout, savedAtStr)
		if err != nil {
			return nil, fmt.Errorf("store: parse saved_at: %w", err)
		}
		m.SavedAt = t
		moments = append(moments, m)
	}
	return moments, rows.Err()
}

// DeleteMoment removes a saved moment by its ID.
func (s *Store) DeleteMoment(id int64) error {
	_, err := s.db.Exec(`DELETE FROM saved_moments WHERE id = ?`, id)
	return err
}
