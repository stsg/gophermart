package store

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type PGStore struct {
	db *sql.DB
}

func NewStore(DBURI string) (*PGStore, error) {
	db, err := sql.Open("postgres", DBURI)
	if err != nil {
		log.Printf("[ERROR] failed to connect to database, %v", err)
		return nil, fmt.Errorf("DB open error: %s", err)
	}

	if !IsTableExist(db, "users") {
		driver, err := postgres.WithInstance(db, &postgres.Config{})
		if err != nil {
			log.Printf("[ERROR] failed to create migration driver, %v", err)
			return nil, fmt.Errorf("failed to create migration driver: %s", err)
		}
		m, err := migrate.NewWithDatabaseInstance(
			"file://data/db/migration",
			"postgres", driver,
		)
		if err != nil {
			log.Printf("[ERROR] DB migrate registration error, %v", err)
			return nil, fmt.Errorf("DB migrate registration error: %s", err)
		}
		err = m.Up()
		if err != nil {
			log.Printf("[ERROR] DB migration error, %v", err)
			return nil, fmt.Errorf("DB migration error: %s", err)
		}
	}

	return &PGStore{db: db}, nil
}

func (s *PGStore) Close() {
	s.db.Close()
}

func (s *PGStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

func (s *PGStore) Execute(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}
func IsTableExist(db *sql.DB, table string) bool {
	var n int64
	query := "SELECT 1 FROM information_schema.tables WHERE table_name = $1"
	err := db.QueryRow(query, table).Scan(&n)
	return err == nil
}
