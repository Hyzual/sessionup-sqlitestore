package sqlitestore

//TODO: use _test package ?

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
)

var expectedDiskError = errors.New("Disk error")

func TestCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("did not expect an error while opening a stub database connection, got one %v", err)
	}
	defer db.Close()
	store := SqliteStore{db: db, tableName: "sessions"}

	query := "INSERT INTO sessions"
	session := sessionup.Session{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now(),
		ID:        "id",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	session.Agent.OS = "GNU/Linux"
	session.Agent.Browser = "Firefox"

	tests := map[string]struct {
		Expect        func()
		ExpectedError error
	}{
		"should return Duplicate ID error": {
			Expect: func() {
				mock.ExpectExec(query).WithArgs(
					session.CreatedAt,
					session.ExpiresAt,
					session.ID,
					session.UserKey,
					session.IP.String(),
					session.Agent.OS,
					session.Agent.Browser,
				).WillReturnError(sqlite3.Error{
					ExtendedCode: sqlite3.ErrConstraintUnique,
				})
			},
			ExpectedError: sessionup.ErrDuplicateID,
		},
		"should return other kinds of error": {
			Expect: func() {
				mock.ExpectExec(query).WithArgs(
					session.CreatedAt,
					session.ExpiresAt,
					session.ID,
					session.UserKey,
					session.IP.String(),
					session.Agent.OS,
					session.Agent.Browser,
				).WillReturnError(expectedDiskError)
			},
			ExpectedError: expectedDiskError,
		},
		"successful create": {
			Expect: func() {
				mock.ExpectExec(query).WithArgs(
					session.CreatedAt,
					session.ExpiresAt,
					session.ID,
					session.UserKey,
					session.IP.String(),
					session.Agent.OS,
					session.Agent.Browser,
				).WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
	}

	for testName, testDefinition := range tests {
		t.Run(testName, func(t *testing.T) {
			testDefinition.Expect()
			err := store.Create(context.Background(), session)
			if err != testDefinition.ExpectedError {
				t.Errorf("want %v, got %v", testDefinition.ExpectedError, err)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}
