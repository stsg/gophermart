package store

import (
	"database/sql"
	"fmt"
)

var (
	ErrUniqueViolation = fmt.Errorf("unique violation")
	ErrNoExists        = fmt.Errorf("no exists")
)

type Store interface {
	Close()
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Execute(query string, args ...interface{}) (sql.Result, error)
}
