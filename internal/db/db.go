package db

import (
	"database/sql"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func MustCreateClient() *sql.DB {
	db, err := sql.Open("postgres", "postgres://postgres:postgresDownscaler@downscaler.cltniyufoxek.us-east-1.rds.amazonaws.com/downscaler")
	// db, err := sql.Open("sqlite", "/home/nonroot/data_db")
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	return db
}
