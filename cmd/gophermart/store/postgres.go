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

	"github.com/stsg/gophermart/cmd/gophermart/lib"
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

	if !lib.IsTableExist(pool, "users") {
		if err := migrate(pool, cfg.MigrationVersion); err != nil {
			return nil, err
		}
	}

	return &Storage{cfg: cfg, db: pool}, nil
}

func (p *Storage) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	var user models.User

	err := p.db.QueryRow(ctx, "SELECT uid, login, password FROM users WHERE login=$1", login).Scan(
		&user.UID,
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

func (p *Storage) GetUserByUUID(ctx context.Context, uid uuid.UUID) (models.User, error) {
	var user models.User

	err := p.db.QueryRow(ctx, "SELECT uid, login, password FROM users WHERE uid=$1", uid).Scan(
		&user.UID,
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
	_, err := p.db.Exec(ctx, "INSERT INTO users (uid, login, password) VALUES ($1, $2, $3)", user.UID, user.Login, user.PHash)
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
		order.ID, user.UID, order.UploadedAt)
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

func (p *Storage) UpdateOrderStatus(ctx context.Context, orderNumber string, status models.AccrualStatus, amount int64) (*models.OrderResponse, error) {
	var uid uuid.UUID

	order := models.OrderResponse{}
	err := p.db.QueryRow(
		ctx,
		"UPDATE orders SET status=$2, amount=$3 WHERE id=$1 RETURNING id, uid, amount, status, updated_at",
		orderNumber, status, amount,
	).Scan(&order.ID, &uid, &order.Amount, &order.Status, &order.UploadedAt)
	if err != nil {
		log.Printf("[ERROR] cannot update order %s status %v", orderNumber, err)
		return &order, err
	}

	// TODO: check err
	user, _ := p.GetUserByUUID(ctx, uid)
	order.Username = user.Login

	return &order, nil
}

func (p *Storage) GetOrders(ctx context.Context, uid uuid.UUID) []models.OrderResponse {
	var orders []models.OrderResponse
	rows, err := p.db.Query(ctx, "SELECT id, uid, amount, status, updated_at FROM orders WHERE uid=$1", uid)
	if err != nil {
		log.Printf("[ERROR] cannot get orders %v", err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var amount int64
		order := models.OrderResponse{}
		err := rows.Scan(&order.ID, &uid, &amount, &order.Status, &order.UploadedAt)
		if err != nil {
			log.Printf("[ERROR] cannot get order %v", err)
			continue
		}
		order.Amount = lib.RoundFloat(float64(amount)/100.00, 2)
		orders = append(orders, order)
	}
	return orders
}

func (p *Storage) GetBalance(ctx context.Context, uid uuid.UUID) models.BalanceResponse {
	var current_balance int64
	var withdrawn int64
	var balance models.BalanceResponse

	err := p.db.QueryRow(ctx, "SELECT current_balance, withdrawn FROM balances WHERE uid=$1", uid).Scan(&current_balance, &withdrawn)
	if err != nil {
		log.Printf("[ERROR] cannot get balance %v", err)
		return models.BalanceResponse{}
	}

	balance.Current = lib.RoundFloat(float64(current_balance)/100.00, 2)
	balance.Withdrawn = lib.RoundFloat(float64(withdrawn)/100.00, 2)

	return balance
}

func (p *Storage) SaveWithdraw(ctx context.Context, user *models.User, order *models.Order) (err error) {
	balance := models.Balance{}
	// accrual := models.Accrual{}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		log.Printf("[ERROR] cannot begin tx %v", err)
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		"UPDATE balances SET current_balance=current_balance-$1, withdrawn=withdrawn+$1 WHERE current_balance>=$1 AND uid=$2 RETURNING uid, current_balance, withdrawn",
		order.Amount, user.UID,
	).Scan(&balance.UID, &balance.Current, &balance.Withdrawn)
	if err != nil {
		log.Printf("[ERROR] cannot update balance for %s status %v", order.ID, err)
		return models.ErrBalanceWrong
	}

	_, err = tx.Exec(
		ctx,
		"INSERT INTO orders (id, uid, amount, status) VALUES ($1, $2, $3, $4)",
		order.ID, user.UID, -order.Amount, models.AccrualStatusProcessed,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			log.Printf("[ERROR] user %s already exists %v", user.Login, err)
			return models.ErrOrderExists
		}
		log.Printf("[ERROR] cannot save order %s %v", user.Login, err)
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Storage) GetWithdrawals(ctx context.Context, uid uuid.UUID) ([]models.WithdrawalsResponse, error) {
	var orders []models.WithdrawalsResponse

	rows, err := p.db.Query(ctx, "SELECT id, amount, updated_at FROM orders WHERE uid=$1 AND amount < 0", uid)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("[ERROR] no orders %v", err)
		return []models.WithdrawalsResponse{}, models.ErrOrderNotFound
	}
	if err != nil {
		log.Printf("[ERROR] cannot get orders %v", err)
		return []models.WithdrawalsResponse{}, err
	}
	defer rows.Close()
	for rows.Next() {
		order := models.WithdrawalsResponse{}
		err := rows.Scan(&order.Number, &order.Accrual, &order.ProcessedAt)
		if err != nil {
			log.Printf("[ERROR] cannot get order %v", err)
			continue
		}
		orders = append(orders, order)
	}
	return orders, err
}

func (p *Storage) GetOrdersByStatus(ctx context.Context, status models.AccrualStatus) ([]models.OrderResponse, error) {
	var orders []models.OrderResponse
	rows, err := p.db.Query(
		ctx,
		"SELECT id, uid, amount, status, updated_at FROM orders WHERE status=$1 and amount > 0 ORDER BY updated_at", status,
	)
	if err != nil {
		log.Printf("[ERROR] cannot get orders %v", err)
		return orders, err
	}
	defer rows.Close()
	for rows.Next() {
		order := models.OrderResponse{}
		err := rows.Scan(&order.ID, &order.Amount, &order.Status, &order.UploadedAt)
		if err != nil {
			log.Printf("[ERROR] cannot get order %v", err)
			continue
		}
		orders = append(orders, order)
	}
	return orders, err
}
