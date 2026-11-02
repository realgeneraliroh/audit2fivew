// Copyright (c) 2026 J
// All rights reserved.
// Released under the GNU Affero General Public License v3.0.

// Package store provides a lightweight SQLite-backed persistence layer
// for historical AVC denial records.
package store

import (
	"database/sql"
	"strings"
	"time"

	// Pure-Go SQLite driver — no CGO required, enabling fully static builds.
	_ "modernc.org/sqlite"

	"audit2fivew/internal/avc"
)

// Store wraps a SQLite database for AVC denial persistence.
type Store struct {
	db *sql.DB
}

// New opens (or creates) the SQLite database at the given path and
// initialises the schema.  The database uses WAL journal mode for
// concurrent-read performance.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Pragmas for reliability and performance.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, err
		}
	}

	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// init creates the denials table and indices if they do not already exist.
func (s *Store) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS avc_denials (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   INTEGER NOT NULL,
			serial      TEXT    NOT NULL DEFAULT '',
			raw_log     TEXT    NOT NULL,
			permissions TEXT    NOT NULL,
			tclass      TEXT    NOT NULL DEFAULT '',
			scontext    TEXT    NOT NULL DEFAULT '',
			tcontext    TEXT    NOT NULL DEFAULT '',
			pid         INTEGER NOT NULL DEFAULT 0,
			comm        TEXT    NOT NULL DEFAULT '',
			name        TEXT    NOT NULL DEFAULT '',
			dev         TEXT    NOT NULL DEFAULT '',
			ino         TEXT    NOT NULL DEFAULT '',
			path        TEXT    NOT NULL DEFAULT '',
			permissive  INTEGER NOT NULL DEFAULT 0,
			hash        TEXT    UNIQUE NOT NULL,
			first_seen  INTEGER NOT NULL,
			last_seen   INTEGER NOT NULL,
			count       INTEGER NOT NULL DEFAULT 1
		);
		CREATE INDEX IF NOT EXISTS idx_denials_last_seen ON avc_denials(last_seen);
		CREATE INDEX IF NOT EXISTS idx_denials_hash      ON avc_denials(hash);
	`)
	return err
}

// Upsert inserts a new denial or, if a denial with the same content hash
// already exists, bumps its count and updates last_seen.  This keeps the
// database de-duplicated while tracking recurrence.
func (s *Store) Upsert(d *avc.Denial) error {
	permStr := strings.Join(d.Permissions, ",")
	permissive := boolToInt(d.Permissive)

	_, err := s.db.Exec(`
		INSERT INTO avc_denials
			(timestamp, serial, raw_log, permissions, tclass,
			 scontext, tcontext, pid, comm, name, dev, ino, path,
			 permissive, hash, first_seen, last_seen, count)
		VALUES (?,?,?,?,?, ?,?,?,?,?, ?,?,?,?,?, ?,?,1)
		ON CONFLICT(hash) DO UPDATE SET
			last_seen = MAX(avc_denials.last_seen, excluded.last_seen),
			count     = avc_denials.count + 1,
			raw_log   = excluded.raw_log
	`,
		d.Timestamp.Unix(), d.Serial, d.RawLog, permStr, d.TClass,
		d.SContext, d.TContext, d.PID, d.Comm, d.Name, d.Dev, d.Ino, d.Path,
		permissive, d.ContentHash, d.FirstSeen.Unix(), d.LastSeen.Unix(),
	)
	return err
}

// List returns every stored denial ordered by last_seen descending
// (most recent first).
func (s *Store) List() ([]*avc.Denial, error) {
	rows, err := s.db.Query(`
		SELECT id, timestamp, serial, raw_log, permissions, tclass,
		       scontext, tcontext, pid, comm, name, dev, ino, path,
		       permissive, hash, first_seen, last_seen, count
		FROM avc_denials
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var denials []*avc.Denial
	for rows.Next() {
		d := &avc.Denial{}
		var (
			permStr    string
			permissive int
			ts, first  int64
			last       int64
		)
		if err := rows.Scan(
			&d.ID, &ts, &d.Serial, &d.RawLog, &permStr, &d.TClass,
			&d.SContext, &d.TContext, &d.PID, &d.Comm, &d.Name,
			&d.Dev, &d.Ino, &d.Path, &permissive, &d.ContentHash,
			&first, &last, &d.Count,
		); err != nil {
			return nil, err
		}

		d.Permissions = strings.Split(permStr, ",")
		d.Permissive = permissive == 1
		d.Timestamp = time.Unix(ts, 0)
		d.FirstSeen = time.Unix(first, 0)
		d.LastSeen = time.Unix(last, 0)

		denials = append(denials, d)
	}
	return denials, rows.Err()
}

// Close releases the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

