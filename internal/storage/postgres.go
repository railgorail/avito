package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"railgorail/avito/internal/lib/sl"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Storage struct {
	Db *sqlx.DB
}

func New(dsn string, log *slog.Logger) (*Storage, error) {
	const op = "storage.postgres.New"

	if err := runMigrations(dsn, log); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{Db: db}, nil
}

func MustInit(dsn string, log *slog.Logger) (*sqlx.DB, func()) {
	storage, err := New(dsn, log)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	cleanup := func() {
		if err := storage.Db.Close(); err != nil {
			log.Error("failed to close db", sl.Err(err))
		}
	}

	return storage.Db, cleanup
}

func runMigrations(dsn string, log *slog.Logger) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close migration db", sl.Err(err))
		}
	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	log.Info("migrations applied successfully")
	return nil
}
