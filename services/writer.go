package services

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jtarchie/sqlite-tsdb/sdk"
)

type Writer struct {
	db       *sql.DB
	insert   *sql.Stmt
	filename string
}

func NewWriter(filename string) (*Writer, error) {
	db, err := sql.Open(dbDriverName, fmt.Sprintf("%s?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL", filename))
	if err != nil {
		return nil, fmt.Errorf("could not open sqlite db %q: %w", filename, err)
	}

	_, err = db.Exec(`
		PRAGMA busy_timeout = 5000;
		PRAGMA journal_mode = WAL;
		PRAGMA synchronous = NORMAL;
		PRAGMA wal_autocheckpoint = 0;
		CREATE TABLE IF NOT EXISTS payloads (
			id         INTEGER PRIMARY KEY,
			payload    TEXT NOT NULL,
			timestamp  INT GENERATED ALWAYS AS (payload->'$.timestamp') VIRTUAL,
			value      TEXT GENERATED ALWAYS AS (payload->'$.value') VIRTUAL
		);
		CREATE INDEX IF NOT EXISTS payloads_timestamp ON payloads(timestamp);
		CREATE VIRTUAL TABLE events USING fts5(value, content=payloads, content_rowid=id);
		CREATE TRIGGER payload_insert AFTER INSERT ON payloads BEGIN
  		INSERT INTO events(rowid, value) VALUES (new.id, new.value);
		END;
	`)

	if err != nil {
		return nil, fmt.Errorf("could not run migrations %q: %w", filename, err)
	}

	insert, err := db.Prepare(`INSERT INTO payloads (payload) VALUES (?);`)
	if err != nil {
		return nil, fmt.Errorf("could not create prepared insert statement: %w", err)
	}

	return &Writer{
		db:       db,
		filename: filename,
		insert:   insert,
	}, nil
}

func (s *Writer) Insert(event *sdk.Event) error {
	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("could not marshal event: %w", err)
	}

	_, err = s.insert.Exec(bytes)
	if err != nil {
		return fmt.Errorf("could not insert payload: %w", err)
	}

	return nil
}

func (s *Writer) Close() error {
	err := s.insert.Close()
	if err != nil {
		return fmt.Errorf("cannot close insert prepared statement: %w", err)
	}

	_, err = s.db.Exec(`
		PRAGMA JOURNAL_MODE = DELETE; -- to be able to actually set page size
		PRAGMA PAGE_SIZE = 1024;      -- trade off of number of requests that need to be made vs overhead.
		INSERT INTO events(events) values ('optimize');
		VACUUM;
	`)
	if err != nil {
		return fmt.Errorf("cannot optimize the database: %w", err)
	}

	err = s.db.Close()
	if err != nil {
		return fmt.Errorf("cannot close database statement: %w", err)
	}

	return nil
}

func (s *Writer) Filename() string {
	return s.filename
}
