package sqlitestore

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
)

const createTableQuery = `CREATE TABLE IF NOT EXISTS %s (
	created_at DATETIME NOT NULL,
	expires_at DATETIME NOT NULL,
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
	//TODO: create channels for cleanup and stuff
	return store, nil
}

// Create implements sessionup.Store interface's Create method.
func (store *SqliteStore) Create(ctx context.Context, session sessionup.Session) error {
	query := fmt.Sprintf("INSERT INTO %s VALUES ($1, $2, $3, $4, $5, $6, $7);", store.tableName)
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
	sqliteError, ok := err.(sqlite3.Error)
	if ok && sqliteError.ExtendedCode == sqlite3.ErrConstraintUnique {
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

// FetchByID implements sessionup.Store interface's FetchByID method.
func (store *SqliteStore) FetchByID(ctx context.Context, id string) (sessionup.Session, bool, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1 AND expires_at > datetime('now', 'localtime')", store.tableName)
	row := store.db.QueryRowContext(ctx, query, id)

	var session sessionup.Session
	var ip, os, browser sql.NullString

	err := row.Scan(&session.CreatedAt, &session.ExpiresAt, &session.ID, &session.UserKey, &ip, &os, &browser)
	if err == sql.ErrNoRows {
		return sessionup.Session{}, false, nil
	} else if err != nil {
		return sessionup.Session{}, false, err
	}

	if ip.Valid {
		session.IP = net.ParseIP(ip.String)
	}

	session.Agent.OS = os.String
	session.Agent.Browser = browser.String
	return session, true, nil
}
