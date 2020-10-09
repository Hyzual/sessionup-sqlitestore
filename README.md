# sessionup-sqlitestore

<!-- TODO: [![Build status]()]() -->
[![Go Report Card](https://goreportcard.com/badge/github.com/Hyzual/sessionup-sqlitestore)](https://goreportcard.com/report/github.com/Hyzual/sessionup-sqlitestore)
[![GoDoc](https://godoc.org/github.com/Hyzual/sessionup-sqlitestore?status.png)](https://godoc.org/github.com/Hyzual/sessionup-sqlitestore)

SQLite session store implementation for [sessionup](https://github.com/swithek/sessionup)

## Installation
```sh
go get github.com/Hyzual/sessionup-sqlitestore
```

## Usage
```go
db, err := sql.Open("sqlite3", "...")
if err != nil {
    // handle error
}

store, err := sqlitestore.New(db, "sessions", time.Minute * 5)
if err != nil {
    // handle error
}

manager := sessionup.NewManager(store)
```

TODO: Links to continue:
https://stackoverflow.com/questions/25965584/separating-unit-tests-and-integration-tests-in-go

https://github.com/swithek/sessionup-pgstore/blob/master/pgstore.go
https://github.com/swithek/sessionup-pgstore/blob/master/pgstore_test.go

https://github.com/mattn/go-sqlite3/blob/b4f5cc77d1cca1470922e916c9f775ef17d2d78f/error_test.go
https://github.com/mattn/go-sqlite3/blob/b4f5cc77d1cca1470922e916c9f775ef17d2d78f/error_test.go
