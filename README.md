# sessionup-sqlitestore

[![CI pipeline](https://github.com/Hyzual/sessionup-sqlitestore/workflows/CI%20pipeline/badge.svg)](https://github.com/Hyzual/sessionup-sqlitestore/actions)
[![codecov](https://codecov.io/gh/Hyzual/sessionup-sqlitestore/branch/master/graph/badge.svg?token=4UXTFPWW41)](https://codecov.io/gh/Hyzual/sessionup-sqlitestore)
[![Go Report Card](https://goreportcard.com/badge/github.com/Hyzual/sessionup-sqlitestore)](https://goreportcard.com/report/github.com/Hyzual/sessionup-sqlitestore)
[![GoDoc](https://godoc.org/github.com/Hyzual/sessionup-sqlitestore?status.png)](https://godoc.org/github.com/Hyzual/sessionup-sqlitestore)

SQLite session store implementation for [sessionup](https://github.com/swithek/sessionup)

## Installation
```sh
go get github.com/hyzual/sessionup-sqlitestore
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
