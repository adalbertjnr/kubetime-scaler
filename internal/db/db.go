package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func MustCreateClient() *sql.DB {
	db, err := sql.Open("sqlite", "/home/nonroot/data_db")
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	return db
}
