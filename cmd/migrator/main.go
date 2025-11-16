package main

import (
	"flag"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsDir = "file://migrations"

func main() {
	// hardcode :)
	dsn := "postgres://someuser:somepassword@localhost:5432/somedb?sslmode=disable"
	m, err := migrate.New(migrationsDir, dsn)
	if err != nil {
		log.Fatalf("Error initializing migrations: %v", err)
	}

	flag.Usage = func() {
		log.Println("Usage:")
		log.Println("  migrator up        - Apply all migrations")
		log.Println("  migrator down      - Rollback the last migration")
	}

	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)
	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Error applying migrations: %v", err)
		}
		log.Println("All migrations applied successfully.")

	case "down":
		if err := m.Down(); err != nil {
			log.Fatalf("Error rolling back migration: %v", err)
		}
		log.Println("Last migration rolled back.")

	default:
		flag.Usage()
		os.Exit(1)
	}
}
