package sqlitestore_test

import (
	"context"
	"database/sql"
	"net"
	"testing"
	"time"

	sqlitestore "github.com/hyzual/sessionup-sqlitestore"
	_ "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
)

func TestSessionByIDIntegration(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:database.db?mode=memory")
	if err != nil {
		db.Close()
		t.Fatalf("could not open in-memory database: %v", err)
	}
	defer db.Close()

	store, err := sqlitestore.New(db, "sessions")
	if err != nil {
		t.Fatalf("could not create a new sessions table: %v", err)
	}

	session := sessionup.Session{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 1),
		ID:        "id",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	ctx := context.Background()
	err = store.Create(ctx, session)
	if err != nil {
		t.Fatalf("could not create a session: %v", err)
	}

	retrievedSession, ok, err := store.FetchByID(ctx, "id")
	if err != nil {
		t.Fatalf("unexpected error while fetching the session by its ID: %v", err)
	}
	if !ok {
		t.Fatalf("expected to find session by its ID, but it was not found")
	}
	assertSessionEquals(t, retrievedSession, session)
}

func TestSessionsByUserKeyIntegration(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:database.db?mode=memory")
	if err != nil {
		db.Close()
		t.Fatalf("could not open in-memory database: %v", err)
	}
	defer db.Close()

	store, err := sqlitestore.New(db, "sessions")
	if err != nil {
		t.Fatalf("could not create a new sessions table: %v", err)
	}

	validSession := sessionup.Session{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 1),
		ID:        "valid",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	expiredSession := sessionup.Session{
		CreatedAt: time.Now().Add(time.Hour * -2),
		ExpiresAt: time.Now().Add(time.Hour * -1),
		ID:        "expired",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	ctx := context.Background()
	err = store.Create(ctx, validSession)
	if err != nil {
		t.Fatalf("could not create a session: %v", err)
	}
	err = store.Create(ctx, expiredSession)
	if err != nil {
		t.Fatalf("could not create a session: %v", err)
	}

	actualSessions, err := store.FetchByUserKey(ctx, "key")
	if err != nil {
		t.Fatalf("unexpected error while fetching the sessions by user key: %v", err)
	}
	if actualSessions == nil {
		t.Fatalf("expected to find sessions by their key, but they were not found")
	}
	assertSessionsContains(t, validSession, actualSessions)
	assertSessionsContains(t, expiredSession, actualSessions)
}

func assertSessionEquals(t *testing.T, actual sessionup.Session, expected sessionup.Session) {
	t.Helper()

	if actual.ID != expected.ID {
		t.Errorf("got ID %s, want %s", actual.ID, expected.ID)
	}
	if !expected.CreatedAt.Equal(actual.CreatedAt) {
		t.Errorf("got CreatedAt %s, want %s", actual.CreatedAt.String(), expected.CreatedAt.String())
	}
	if !expected.ExpiresAt.Equal(actual.ExpiresAt) {
		t.Errorf("got ExpiresAt %s, want %s", actual.CreatedAt.String(), expected.CreatedAt.String())
	}
	if !expected.IP.Equal(actual.IP) {
		t.Errorf("got IP %s, want %s", actual.IP.String(), expected.IP.String())
	}
	if actual.UserKey != expected.UserKey {
		t.Errorf("got UserKey %s, want %s", actual.UserKey, expected.UserKey)
	}
	if actual.Agent.OS != expected.Agent.OS {
		t.Errorf("got Agent.OS %s, want %s", actual.Agent.OS, expected.Agent.OS)
	}
	if actual.Agent.Browser != expected.Agent.Browser {
		t.Errorf("got Agent.Browser %s, want %s", actual.Agent.Browser, expected.Agent.Browser)
	}
}

func sessionEquals(actual, expected sessionup.Session) bool {
	if !expected.CreatedAt.Equal(actual.CreatedAt) {
		return false
	}
	if !expected.ExpiresAt.Equal(actual.ExpiresAt) {
		return false
	}
	if !expected.IP.Equal(actual.IP) {
		return false
	}
	return actual.ID == expected.ID &&
		actual.UserKey == expected.UserKey &&
		actual.Agent.OS == expected.Agent.OS &&
		actual.Agent.Browser == expected.Agent.Browser
}

func assertSessionsContains(t *testing.T, needle sessionup.Session, haystack []sessionup.Session) {
	t.Helper()

	for _, current := range haystack {
		if sessionEquals(current, needle) {
			return
		}
	}

	t.Errorf("could not find session %v among sessions %v", needle, haystack)
}
