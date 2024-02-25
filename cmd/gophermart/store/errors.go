package postgres

import (
	"fmt"
)

var (
	ErrUniqueViolation = fmt.Errorf("unique violation")
	ErrNoExists        = fmt.Errorf("no exists")
)
