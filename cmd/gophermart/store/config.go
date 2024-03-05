package postgres

import (
	"time"
)

type Config struct {
	ConnectionString string
	ConnectTimeout   time.Duration
	QueryTimeout     time.Duration
	MigrationVersion int64
}
