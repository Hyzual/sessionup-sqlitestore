package sqlitestore

import (
	"context"
	"database/sql"
	"fmt"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
)

const createTableQuery = `CREATE TABLE IF NOT EXISTS %s (
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	id TEXT PRIMARY KEY,
	user_key TEXT NOT NULL,
	ip TEXT,
	agent_os TEXT,
	agent_browser TEXT
);`

type SqliteStore struct {
	db        *sql.DB
	tableName string
}

func New(db *sql.DB, tableName string) (*SqliteStore, error) {
	store := &SqliteStore{db: db, tableName: tableName}
	_, err := store.db.Exec(fmt.Sprintf(createTableQuery, store.tableName))
	if err != nil {
		return nil, err
	}
	return store, nil
}

// Create implements sessionup.Store interface's Create method.
func (store *SqliteStore) Create(ctx context.Context, session sessionup.Session) error {
	query := fmt.Sprintf("INSERT INTO %s VALUES (?, ?, ?, ?, ?, ?, ?);", store.tableName)
	_, err := store.db.ExecContext(
		ctx,
		query,
		session.CreatedAt,
		session.ExpiresAt,
		session.ID,
		session.UserKey,
		wrapNullString(session.IP.String()),
		wrapNullString(session.Agent.OS),
		wrapNullString(session.Agent.Browser),
	)
	if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
		return sessionup.ErrDuplicateID
	}
	return err
}

// wrapNullString wraps the given string into an sql.NullString.
func wrapNullString(s string) sql.NullString {
	var nullString sql.NullString
	if s != "" && s != "<nil>" {
		nullString.String = s
		nullString.Valid = true
	}
	return nullString
}
