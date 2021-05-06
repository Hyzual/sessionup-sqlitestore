/*
Package sqlitestore implements sessionup.Store interface for SQLite database.
*/
package sqlitestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

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
	agent_browser TEXT,
	metadata TEXT
);`

const (
	PART_SEPARATOR      = ";"
	KEY_VALUE_SEPARATOR = ":"
)

// SqliteStore is a SQLite implementation of sessionup.Store.
type SqliteStore struct {
	db        *sql.DB
	tableName string
	stopChan  chan struct{}
	errChan   chan error
}

// New returns a fresh instance of SqliteStore.
// tableName parameter determines the name of the table that will be used for
// sessions. If it does not exist, it will be created.
// Duration parameter determines how often the cleanup function wil be called
// to remove the expired sessions. Setting it to 0 will prevent cleanup from
// being activated.
func New(db *sql.DB, tableName string, duration time.Duration) (*SqliteStore, error) {
	store := &SqliteStore{db: db, tableName: tableName, errChan: make(chan error)}
	_, err := store.db.Exec(fmt.Sprintf(createTableQuery, store.tableName))
	if err != nil {
		return nil, err
	}

	if duration > 0 {
		go store.startCleanup(duration)
	}
	return store, nil
}

// Create implements sessionup.Store interface's Create method.
func (store *SqliteStore) Create(ctx context.Context, session sessionup.Session) error {
	query := fmt.Sprintf("INSERT INTO %s VALUES ($1, $2, $3, $4, $5, $6, $7, $8);", store.tableName)
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
		serializeMetadata(session.Meta),
	)
	var sqliteError sqlite3.Error
	if errors.As(err, &sqliteError) && sqliteError.Code == sqlite3.ErrConstraint {
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
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1 AND expires_at > datetime('now', 'localtime');", store.tableName) // nolint:gosec // Concatenation is used for table name, not bound parameters
	row := store.db.QueryRowContext(ctx, query, id)

	var session sessionup.Session
	var ip, os, browser, metadata sql.NullString

	err := row.Scan(&session.CreatedAt, &session.ExpiresAt, &session.ID, &session.UserKey, &ip, &os, &browser, &metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return sessionup.Session{}, false, nil
	} else if err != nil {
		return sessionup.Session{}, false, err
	}

	if ip.Valid {
		session.IP = net.ParseIP(ip.String)
	}

	session.Agent.OS = os.String
	session.Agent.Browser = browser.String
	session.Meta = parseMetadata(metadata)
	return session, true, nil
}

// FetchByUserKey implements sessionup.Store interface's FetchByUserKey method.
func (store *SqliteStore) FetchByUserKey(ctx context.Context, key string) ([]sessionup.Session, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE user_key = $1;", store.tableName) // nolint:gosec // Concatenation is used for table name, not bound parameters
	rows, err := store.db.QueryContext(ctx, query, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var foundSessions []sessionup.Session
	for rows.Next() {
		var session sessionup.Session
		var ip, os, browser, metadata sql.NullString

		err := rows.Scan(&session.CreatedAt, &session.ExpiresAt, &session.ID, &session.UserKey, &ip, &os, &browser, &metadata)
		if err != nil {
			defer rows.Close()
			return nil, err
		}

		if ip.Valid {
			session.IP = net.ParseIP(ip.String)
		}

		session.Agent.OS = os.String
		session.Agent.Browser = browser.String
		session.Meta = parseMetadata(metadata)

		foundSessions = append(foundSessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return foundSessions, nil
}

// serializeMetadata converts metadata map of string to a string to be saved in DB.
func serializeMetadata(source map[string]string) sql.NullString {
	var builder strings.Builder
	for key, value := range source {
		builder.WriteString(fmt.Sprintf("%s:%s;", key, value))
	}
	return wrapNullString(builder.String())
}

// parseMetadata converts metadata string from DB into a map of strings.
func parseMetadata(source sql.NullString) map[string]string {
	if !source.Valid {
		return nil
	}

	meta := make(map[string]string)
	parts := strings.Split(source.String, PART_SEPARATOR)
	for _, part := range parts {
		keyValue := strings.Split(part, KEY_VALUE_SEPARATOR)
		if len(keyValue) != 2 {
			continue
		}

		meta[keyValue[0]] = keyValue[1]
	}
	return meta
}

// DeleteByID implements sessionup.Store interface's DeleteByID method.
func (store *SqliteStore) DeleteByID(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1;", store.tableName)
	_, err := store.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByUserKey implements sessionup.Store interface's DeleteByUserKey method.
func (store *SqliteStore) DeleteByUserKey(ctx context.Context, key string, sessionIDsToKeep ...string) error {
	if len(sessionIDsToKeep) > 0 {
		params := make([]interface{}, 0)
		params = append(params, key)
		for _, id := range sessionIDsToKeep {
			params = append(params, id)
		}
		query := fmt.Sprintf("DELETE FROM %s WHERE user_key = $1 AND id NOT IN (?"+strings.Repeat(",?", len(params)-2)+");", store.tableName)
		_, err := store.db.ExecContext(ctx, query, params...)
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE user_key = $1;", store.tableName)
	_, err := store.db.ExecContext(ctx, query, key)
	return err
}

// deleteExpired deletes all expired sessions.
func (store *SqliteStore) deleteExpired() error {
	query := fmt.Sprintf("DELETE FROM %s WHERE expires_at < datetime('now', 'localtime');", store.tableName)
	_, err := store.db.Exec(query)
	return err
}

func (store *SqliteStore) startCleanup(duration time.Duration) {
	store.stopChan = make(chan struct{})
	timer := time.NewTicker(duration)
	for {
		select {
		case <-timer.C:
			if err := store.deleteExpired(); err != nil {
				store.errChan <- err
			}

		case <-store.stopChan:
			timer.Stop()
			return
		}
	}
}

// StopCleanup terminates the automatic cleanup process.
// Useful for testing and cases when store is used only temporarily.
// In order to restart the cleanup, a new store must be created.
func (store *SqliteStore) StopCleanup() {
	if store.stopChan != nil {
		store.stopChan <- struct{}{}
	}
}

// CleanupErr returns a receive-only channel to get errors produced during the
// automatic cleanup.
// NOTE: channel must be drained in order for the cleanup process to be able to
// continue.
func (store *SqliteStore) CleanupErr() <-chan error {
	return store.errChan
}
