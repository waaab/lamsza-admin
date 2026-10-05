package db

import (
	"backend/internal/config"
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	connStr := config.AppConfig.DatabaseURL

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Database connection established")
	dropLocationsLegacy()
}

func dropLocationsLegacy() {
	var exists bool
	if err := DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'locations_legacy'
		)`).Scan(&exists); err != nil {
		log.Printf("locations_legacy check: %v", err)
		return
	}
	if !exists {
		return
	}
	if _, err := DB.Exec(`DROP TABLE locations_legacy`); err != nil {
		log.Printf("drop locations_legacy: %v", err)
		return
	}
	log.Println("Dropped leftover table locations_legacy")
}
