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

	store, err := sqlitestore.New(db, "sessions", 0)
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
	sessions := []sessionup.Session{validSession, expiredSession}
	for _, s := range sessions {
		err = store.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("could not create a session: %v", err)
		}
	}

	retrievedSession, ok, err := store.FetchByID(context.Background(), "valid")
	if err != nil {
		t.Fatalf("unexpected error while fetching the session by its ID: %v", err)
	}
	if !ok {
		t.Fatalf("expected to find session by its ID, but it was not found")
	}
	assertSessionEquals(t, retrievedSession, validSession)

	err = store.DeleteByID(context.Background(), "valid")
	if err != nil {
		t.Fatalf("unexpected error while deleting the session by its ID: %v", err)
	}

	_, ok, err = store.FetchByID(context.Background(), "valid")
	if err != nil {
		t.Fatalf("unexpected error while fetching again the session by its ID: %v", err)
	}
	if ok {
		t.Fatalf("expected to no longer find the session by its ID after deleting it, but it was found")
	}

	_, ok, err = store.FetchByID(context.Background(), "expired")
	if err != nil {
		t.Fatalf("unexpected error while fetching the expired session by its ID: %v", err)
	}
	if ok {
		t.Fatalf("expected not to find an expired session by its ID, but it was found")
	}
}

func TestSessionsByUserKeyIntegration(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:database.db?mode=memory")
	if err != nil {
		db.Close()
		t.Fatalf("could not open in-memory database: %v", err)
	}
	defer db.Close()

	store, err := sqlitestore.New(db, "sessions", 0)
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
	sessionToBeDeleted := sessionup.Session{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 1),
		ID:        "to_be_deleted",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.01"),
	}
	sessions := []sessionup.Session{validSession, expiredSession, sessionToBeDeleted}

	for _, s := range sessions {
		err = store.Create(context.Background(), s)
		if err != nil {
			t.Fatalf("could not create a session: %v", err)
		}
	}

	actualSessions, err := store.FetchByUserKey(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error while fetching the sessions by user key: %v", err)
	}
	if actualSessions == nil {
		t.Fatalf("expected to find sessions by their key, but they were not found")
	}
	assertSessionsContains(t, validSession, actualSessions)
	assertSessionsContains(t, expiredSession, actualSessions)
	assertSessionsContains(t, sessionToBeDeleted, actualSessions)

	err = store.DeleteByUserKey(context.Background(), "key", "valid", "expired")
	if err != nil {
		t.Fatalf("unexpected error while deleting sessions with exceptions: %v", err)
	}
	sessionsAfterDeletion, err := store.FetchByUserKey(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error while fetching again the sessions by user key: %v", err)
	}
	assertSessionsContains(t, validSession, sessionsAfterDeletion)
	assertSessionsContains(t, expiredSession, sessionsAfterDeletion)
	assertSessionsDoesNotContain(t, sessionToBeDeleted, sessionsAfterDeletion)

	err = store.DeleteByUserKey(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error while deleting all sessions by key: %v", err)
	}
	sessionsAfterSecondDeletion, err := store.FetchByUserKey(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error while fetching for the third time sessions by user key: %v", err)
	}
	if len(sessionsAfterSecondDeletion) > 0 {
		t.Fatal("expected sessions to be empty after deletion")
	}
}

func TestExpiredSessionsCleanupIntegration(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:database.db?mode=memory")
	if err != nil {
		db.Close()
		t.Fatalf("could not open in-memory database: %v", err)
	}
	defer db.Close()

	store, err := sqlitestore.New(db, "sessions", time.Millisecond*20)
	if err != nil {
		t.Fatalf("could not create a new sessions table: %v", err)
	}

	firstExpiredSession := sessionup.Session{
		CreatedAt: time.Now().Add(time.Hour * -2),
		ExpiresAt: time.Now().Add(time.Hour * -1),
		ID:        "first",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	secondExpiredSession := sessionup.Session{
		CreatedAt: time.Now().Add(time.Hour * -3),
		ExpiresAt: time.Now().Add(time.Hour * -2),
		ID:        "second",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	sessions := []sessionup.Session{firstExpiredSession, secondExpiredSession}

	createSessions := func() {
		for _, s := range sessions {
			err = store.Create(context.Background(), s)
			if err != nil {
				t.Fatalf("could not create a session: %v", err)
			}
		}
	}

	waitForCleanup := func() {
		// wait for cleanup after 20 ms
		time.Sleep(time.Millisecond * 21)
	}

	createSessions()
	waitForCleanup()
	assertErrorChannelIsEmpty(t, store.CleanupErr())

	retrievedSessions, err := store.FetchByUserKey(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error while fetching the sessions by user key: %v", err)
	}
	if len(retrievedSessions) > 0 {
		t.Fatal("expected sessions to be empty after cleanup")
	}

	createSessions()
	store.StopCleanup()
	waitForCleanup()

	retrievedSessions, err = store.FetchByUserKey(context.Background(), "key")
	if err != nil {
		t.Fatalf("unexpected error while fetching the sessions by user key: %v", err)
	}
	if len(retrievedSessions) == 0 {
		t.Fatal("expected to find expired sessions after stopping cleanup, but none were found")
	}
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

func assertSessionsDoesNotContain(t *testing.T, needle sessionup.Session, haystack []sessionup.Session) {
	t.Helper()

	for _, current := range haystack {
		if sessionEquals(current, needle) {
			t.Errorf("expected not to find session %v among retrieved sessions, but it was found", needle)
			return
		}
	}
}

func assertErrorChannelIsEmpty(t *testing.T, errChan <-chan error) {
	t.Helper()

	for {
		select {
		case err := <-errChan:
			t.Fatalf("unexpected error during cleanup of expired sessions: %v", err)
		default:
			return
		}
	}
}
