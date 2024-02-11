package store

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type PGStore struct {
	DB *sql.DB
}

func NewStore(DBURI string) *PGStore {
	db, err := sql.Open("postgres", DBURI)
	if err != nil {
		log.Fatal(err)
	}

	return &PGStore{DB: db}
}

func (s *PGStore) Close() {
	s.DB.Close()
}

func (s *PGStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.DB.Query(query, args...)
}

func (s *PGStore) Execute(query string, args ...interface{}) (sql.Result, error) {
	return s.DB.Exec(query, args...)
}
