package sqlitestore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/swithek/sessionup"
)

var expectedDiskError = errors.New("Disk error")

func TestCreate(t *testing.T) {
	db, mock := mockDB(t)
	defer db.Close()
	store := SqliteStore{db: db, tableName: "sessions"}

	query := "INSERT INTO sessions VALUES ($1, $2, $3, $4, $5, $6, $7);"
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
			assertExpectationsWereMet(t, mock)
		})
	}
}

func TestFetchByID(t *testing.T) {
	type check func(*testing.T, sessionup.Session, bool, error)

	checks := func(cc ...check) []check { return cc }

	expectNoError := func() check {
		return func(t *testing.T, _ sessionup.Session, _ bool, actual error) {
			assertNoError(t, actual)
		}
	}

	expectAnError := func(expected error) check {
		return func(t *testing.T, _ sessionup.Session, _ bool, actual error) {
			assertError(t, expected, actual)
		}
	}

	assertSessionMatches := func(expected sessionup.Session, expectSessionIsFound bool) check {
		return func(t *testing.T, actual sessionup.Session, actualSessionIsFound bool, _ error) {
			t.Helper()
			if actualSessionIsFound != expectSessionIsFound {
				t.Errorf("want %t, got %t", expectSessionIsFound, actualSessionIsFound)
			}

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("want %v, got %v", expected, actual)
			}
		}
	}

	db, mock := mockDB(t)
	defer db.Close()
	store := SqliteStore{db: db, tableName: "sessions"}

	query := "SELECT * FROM sessions WHERE id = $1 AND expires_at > datetime('now', 'localtime');"
	session := sessionup.Session{
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour * 1),
		ID:        "id",
		UserKey:   "key",
		IP:        net.ParseIP("127.0.0.1"),
	}
	session.Agent.OS = "GNU/Linux"
	session.Agent.Browser = "Firefox"

	tests := map[string]struct {
		Expect func()
		Checks []check
	}{
		"should return found = false when it gets sql.ErrNoRows": {
			Expect: func() {
				mock.ExpectQuery(query).WithArgs(session.ID).WillReturnError(sql.ErrNoRows)
			},
			Checks: checks(
				expectNoError(),
				assertSessionMatches(sessionup.Session{}, false),
			),
		},
		"should return other kinds of error": {
			Expect: func() {
				mock.ExpectQuery(query).WithArgs(session.ID).WillReturnError(expectedDiskError)
			},
			Checks: checks(
				expectAnError(expectedDiskError),
				assertSessionMatches(sessionup.Session{}, false),
			),
		},
		"should return a Session and found = true": {
			Expect: func() {
				rows := sqlmock.NewRows([]string{"created_at", "expires_at", "id", "user_key", "ip", "agent_os", "agent_browser"}).
					AddRow(session.CreatedAt, session.ExpiresAt, session.ID, session.UserKey, session.IP.String(), session.Agent.OS, session.Agent.Browser)
				mock.ExpectQuery(query).WithArgs(session.ID).WillReturnRows(rows)
			},
			Checks: checks(
				expectNoError(),
				assertSessionMatches(session, true),
			),
		},
	}

	for testName, testDefinition := range tests {
		t.Run(testName, func(t *testing.T) {
			testDefinition.Expect()
			retrievedSession, ok, err := store.FetchByID(context.Background(), session.ID)
			for _, currentCheck := range testDefinition.Checks {
				currentCheck(t, retrievedSession, ok, err)
			}
			assertExpectationsWereMet(t, mock)
		})
	}
}

func TestFetchByUserKey(t *testing.T) {
	type check func(*testing.T, []sessionup.Session, error)

	checks := func(cc ...check) []check { return cc }

	expectNoError := func() check {
		return func(t *testing.T, _ []sessionup.Session, actual error) {
			assertNoError(t, actual)
		}
	}

	expectAnError := func(expected error) check {
		return func(t *testing.T, _ []sessionup.Session, actual error) {
			assertError(t, expected, actual)
		}
	}

	assertSessionsMatch := func(expected []sessionup.Session) check {
		return func(t *testing.T, actual []sessionup.Session, _ error) {
			t.Helper()
			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("want %v, got %v", expected, actual)
			}
		}
	}

	db, mock := mockDB(t)
	defer db.Close()
	key := "key"
	store := SqliteStore{db: db, tableName: "sessions"}

	query := "SELECT * FROM sessions WHERE user_key = $1;"

	generateSessions := func() []sessionup.Session {
		var res []sessionup.Session
		for i := 0; i < 3; i++ {
			res = append(res, sessionup.Session{ID: fmt.Sprintf("id%d", i)})
		}
		return res
	}

	tests := map[string]struct {
		Expect func()
		Checks []check
	}{
		"should return nil when it gets sql.ErrNoRows": {
			Expect: func() {
				mock.ExpectQuery(query).WithArgs(key).WillReturnError(sql.ErrNoRows)
			},
			Checks: checks(
				expectNoError(),
				assertSessionsMatch(nil),
			),
		},
		"should return other kinds of error": {
			Expect: func() {
				mock.ExpectQuery(query).WithArgs(key).WillReturnError(expectedDiskError)
			},
			Checks: checks(
				expectAnError(expectedDiskError),
				assertSessionsMatch(nil),
			),
		},
		"should return found sessions": {
			Expect: func() {
				rows := sqlmock.NewRows([]string{"created_at", "expires_at", "id", "user_key", "ip", "agent_os", "agent_browser"})
				for _, session := range generateSessions() {
					rows.AddRow(session.CreatedAt, session.ExpiresAt, session.ID, session.UserKey, session.IP, session.Agent.OS, session.Agent.Browser)
				}
				mock.ExpectQuery(query).WithArgs(key).WillReturnRows(rows)
			},
			Checks: checks(
				expectNoError(),
				assertSessionsMatch(generateSessions()),
			),
		},
	}

	for testName, testDefinition := range tests {
		t.Run(testName, func(t *testing.T) {
			testDefinition.Expect()
			retrievedSessions, err := store.FetchByUserKey(context.Background(), key)
			for _, currentCheck := range testDefinition.Checks {
				currentCheck(t, retrievedSessions, err)
			}
			assertExpectationsWereMet(t, mock)
		})
	}
}

func TestDeleteByID(t *testing.T) {
	db, mock := mockDB(t)
	defer db.Close()
	id := "id"
	store := SqliteStore{db: db, tableName: "sessions"}
	query := "DELETE FROM sessions WHERE id = $1;"

	t.Run("when there is an error, it should return it", func(t *testing.T) {
		mock.ExpectExec(query).WithArgs(id).WillReturnError(expectedDiskError)
		err := store.DeleteByID(context.Background(), id)
		if err == nil {
			t.Errorf("expected an error but did not get one")
		}
		assertExpectationsWereMet(t, mock)
	})

	t.Run("deletes the session in DB", func(t *testing.T) {
		mock.ExpectExec(query).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))
		err := store.DeleteByID(context.Background(), id)
		if err != nil {
			t.Errorf("expected no error but got one: %v", err)
		}
		assertExpectationsWereMet(t, mock)
	})
}

func TestDeleteByUserKey(t *testing.T) {
	db, mock := mockDB(t)
	defer db.Close()
	key := "key"
	ids := []string{"id1", "id2", "id3"}
	store := SqliteStore{db: db, tableName: "sessions"}

	tests := map[string]struct {
		Expect           func()
		SessionIDsToKeep []string
		ExpectedError    error
	}{
		"should return errors when deleting": {
			Expect: func() {
				query := "DELETE FROM sessions WHERE user_key = $1;"
				mock.ExpectExec(query).WithArgs(key).WillReturnError(expectedDiskError)
			},
			ExpectedError: expectedDiskError,
		},
		"deletes all sessions by user key": {
			Expect: func() {
				query := "DELETE FROM sessions WHERE user_key = $1;"
				mock.ExpectExec(query).WithArgs(key).WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		"should return errors when deleting with exceptions": {
			Expect: func() {
				query := "DELETE FROM sessions WHERE user_key = $1 AND id NOT IN (?,?,?);"
				expectedParams := append([]driver.Value{key}, "id1", "id2", "id3")
				mock.ExpectExec(query).WithArgs(expectedParams...).WillReturnError(expectedDiskError)
			},
			SessionIDsToKeep: ids,
			ExpectedError:    expectedDiskError,
		},
		"deletes all sessions except the IDs given in parameter": {
			Expect: func() {
				query := "DELETE FROM sessions WHERE user_key = $1 AND id NOT IN (?,?,?);"
				expectedParams := append([]driver.Value{key}, "id1", "id2", "id3")
				mock.ExpectExec(query).WithArgs(expectedParams...).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			SessionIDsToKeep: ids,
		},
	}

	for testName, testDescription := range tests {
		t.Run(testName, func(t *testing.T) {
			testDescription.Expect()
			err := store.DeleteByUserKey(context.Background(), key, testDescription.SessionIDsToKeep...)
			if err != testDescription.ExpectedError {
				t.Errorf("expected an error %v but got %v", testDescription.ExpectedError, err)
			}
			assertExpectationsWereMet(t, mock)
		})
	}
}

func TestWrapNullString(t *testing.T) {
	s := wrapNullString("")
	if s.Valid {
		t.Errorf("expected empty string to be invalid, but it was valid")
	}
	if s.String != "" {
		t.Errorf("want %q, got %q", "", s.String)
	}

	s = wrapNullString("<nil>")
	if s.Valid {
		t.Errorf("expected <nil> string to be invalid, but it was valid")
	}
	if s.String != "" {
		t.Errorf("want %q, got %q", "", s.String)
	}

	s = wrapNullString("valid string")
	if !s.Valid {
		t.Errorf("expected test string to be valid, but it was not")
	}
	if s.String == "" {
		t.Errorf("want %q, got %q", "valid string", s.String)
	}
}

func mockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("did not expect an error while opening a stub database connection, got one: %v", err)
	}
	return db, mock
}

func assertExpectationsWereMet(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %v", err)
	}
}

func assertError(t *testing.T, expected error, actual error) {
	t.Helper()
	if actual != expected {
		t.Errorf("expected an error %v but got %v", expected, actual)
	}
}

func assertNoError(t *testing.T, actual error) {
	t.Helper()
	if actual != nil {
		t.Errorf("expected no error but got one: %v", actual)
	}
}
