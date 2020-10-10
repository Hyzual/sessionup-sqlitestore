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

func TestSQLiteIntegration(t *testing.T) {
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
	if !ok {
		t.Fatalf("could not find the session by its ID")
	}
	if err != nil {
		t.Fatalf("unexpected error while retrieving the session by its ID: %v", err)
	}
	assertSessionEquals(t, retrievedSession, session)
}

func assertSessionEquals(t *testing.T, actual sessionup.Session, expected sessionup.Session) {
	t.Helper()

	//TODO: use reflect.DeepEqual ?

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
