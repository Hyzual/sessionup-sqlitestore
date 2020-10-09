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
		t.Errorf("could not open in-memory database %v", err)
	}
	defer db.Close()

	store, err := sqlitestore.New(db, "sessions")
	if err != nil {
		t.Errorf("could not create a new sessions table %v", err)
	}

	session := sessionup.Session{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now(),
		ID:        "id",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	ctx := context.Background()
	err = store.Create(ctx, session)
	if err != nil {
		t.Errorf("could not create a session %v", err)
	}
	//TODO: add more operations
}
