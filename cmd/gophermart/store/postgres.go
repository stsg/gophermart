package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stsg/gophermart/cmd/gophermart/models"
)

type Storage struct {
	cfg *Config
	db  *pgxpool.Pool
	// do  func(ctx context.Context, tx pgx.Tx) error
}

func (p *Storage) Close() {
	p.db.Close()
}

func (p *Storage) Ping(ctx context.Context) error {
	return p.db.Ping(ctx)
}

func New(cfg *Config) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	if !IsTableExist(pool, "users") {
		if err := migrate(pool, cfg.MigrationVersion); err != nil {
			return nil, err
		}
	}

	return &Storage{cfg: cfg, db: pool}, nil
}

func (p *Storage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	var user models.User

	err := p.db.QueryRow(ctx, "SELECT uuid, login, password FROM users WHERE login=$1", login).Scan(
		&user.UUID,
		&user.Login,
		&user.PHash,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNoExists
		}
		return nil, err
	}
	return &user, nil

}

func (p *Storage) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (models.User, error) {
	var user models.User

	err := p.db.QueryRow(ctx, "SELECT uuid, login, password FROM users WHERE uuid=$1", uuid).Scan(
		&user.UUID,
		&user.Login,
		&user.PHash,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.User{}, ErrNoExists
		}
		return models.User{}, err
	}
	return user, nil

}

func (p *Storage) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	_, err := p.db.Exec(ctx, "INSERT INTO users (uuid, login, password) VALUES ($1, $2, $3)", user.UUID, user.Login, user.PHash)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			log.Printf("[ERROR] user %s already exists %v", user.Login, err)
			return nil, models.ErrUserExists
		}
		log.Printf("[ERROR] cannot create user %s %v", user.Login, err)
		return nil, err
	}
	return user, nil
}

func (p *Storage) SaveOrder(ctx context.Context, user *models.User, order *models.Order) (*models.Order, error) {
	order.UploadedAt = time.Now()
	_, err := p.db.Exec(ctx, "INSERT INTO orders (id, uid, updated_at) VALUES ($1, $2, $3)",
		order.ID, user.UUID, order.UploadedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			log.Printf("[ERROR] user %s already exists %v", user.Login, err)
			return nil, models.ErrOrderExists
		}
		log.Printf("[ERROR] cannot save order %s %v", user.Login, err)
		return nil, err
	}
	return order, nil
}

func (p *Storage) UpdateOrderStatus(ctx context.Context, orderNumber string, status models.AccrualStatus, accrual int) (*models.OrderResponse, error) {
	var uuid uuid.UUID
	order := models.OrderResponse{}
	err := p.db.QueryRow(
		ctx,
		"UPDATE ORDERS SET STATUS=$1, ACCRUAL=$2 WHERE ID=$2 RETURNING id, uid, number, status, accrual, updated_at",
		orderNumber, status, accrual).Scan(&order.ID, &uuid, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
	if err != nil {
		log.Printf("[ERROR] cannot update order %s status %v", orderNumber, err)
		return &order, err
	}

	// TODO: check err
	user, _ := p.GetUserByUUID(ctx, uuid)
	order.Username = user.Login

	return &order, nil
}

func IsTableExist(p *pgxpool.Pool, table string) bool {
	var n int

	err := p.QueryRow(context.Background(), "SELECT 1 FROM information_schema.tables WHERE table_name = $1", table).Scan(&n)

	return err == nil
}
