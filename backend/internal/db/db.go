package db

import (
	"backend/internal/config"
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB opens the shared Postgres connection and nothing else.
//
// The admin process must never run DDL: it shares one database with the main
// lamsza backend, and that backend owns the schema. Opening the pool here used
// to be followed by a `DROP TABLE locations_legacy`, which let an admin restart
// change a schema it does not own. The main backend still drops that table on
// its own boot path, and `lamsza/backend/migrations/drop_locations_legacy.sql`
// is the explicit, opt-in form of the same statement.
//
// `boot_ddl_test.go` fails if any DDL is reachable from this process's startup
// path again.
func InitDB() {
	connStr := config.AppConfig.DatabaseURL
	if err := guardTestConnection(connStr); err != nil {
		log.Fatal(err)
	}

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Database connection established")
}
