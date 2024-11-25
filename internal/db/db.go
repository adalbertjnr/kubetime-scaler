package db

import (
	"database/sql"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type Config struct {
	Driver string
	DSN    string
}

func MustCreateClient(dbDriver string, c Config) *sql.DB {
	db, err := sql.Open(c.Driver, c.DSN)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		panic(err)
	}

	return db
}
