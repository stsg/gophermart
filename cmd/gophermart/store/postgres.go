package postgres

import (
	"context"
	"errors"
	"fmt"

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

func (p *Storage) GetUserByLogin(ctx context.Context, login string) (models.User, error) {
	var user models.User

	err := p.db.QueryRow(
		ctx,
		"SELECT uid, login, password FROM users WHERE login=$1", login).Scan(
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

func (p *Storage) GetUserByUUID(ctx context.Context, uid uuid.UUID) (models.User, error) {
	var user models.User

	err := p.db.QueryRow(
		ctx,
		"SELECT uid, login, password FROM users WHERE uid=$1", uid).Scan(
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
	_, err := p.db.Exec(
		ctx,
		"INSERT INTO users (uid, login, password) VALUES ($1, $2, $3)",
		user.UID,
		user.Login,
		user.PHash,
	)
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

func (p *Storage) SaveOrder(ctx context.Context, user models.User, order models.Order) (models.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, p.cfg.QueryTimeout)
	defer cancel()

	err := p.db.QueryRow(ctx, "SELECT id, uid, amount, status, updated_at FROM orders WHERE id=$1 LIMIT 1", order.ID).Scan(
		&order.ID,
		&order.UID,
		&order.Amount,
		&order.AccrualStatus,
		&order.UploadedAt,
	)
	if err == nil {
		if user.UID == order.UID {
			log.Printf("[ERROR] order %s already exist for user %s", order.ID, user.Login)
			return order, models.ErrOrderExists
		}
		log.Printf("[ERROR] order %s already exist for another user %s", order.ID, order.UID)
		return order, models.ErrOrderBelongsAnotherUser
	}

	_, err = p.db.Exec(ctx, "INSERT INTO orders (id, uid, amount, status, updated_at) VALUES ($1, $2, $3, $4, $5)",
		order.ID,
		order.UID,
		order.Amount,
		order.AccrualStatus,
		order.UploadedAt,
	)
	if err != nil {
		log.Printf("[ERROR] cannot save order %s %v", user.Login, err)
		return order, err
	}
	return order, nil
}

func (p *Storage) GetOrders(ctx context.Context, uid uuid.UUID) ([]models.OrderResponse, error) {
	var orders []models.OrderResponse

	ctx, cancel := context.WithTimeout(ctx, p.cfg.QueryTimeout)
	defer cancel()

	rows, err := p.db.Query(
		ctx,
		"SELECT id, uid, amount, status, updated_at FROM orders WHERE uid=$1 order by updated_at",
		uid,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return []models.OrderResponse{}, models.ErrOrderNotFound
		}
		log.Printf("[ERROR] cannot get orders %v", err)
		return []models.OrderResponse{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var amount int64
		order := models.OrderResponse{}
		err := rows.Scan(&order.ID, &uid, &amount, &order.Status, &order.UploadedAt)
		if err != nil {
			log.Printf("[ERROR] cannot scan %v", err)
			continue
		}
		order.Amount = lib.RoundFloat(float64(amount)/100.00, 2)
		orders = append(orders, order)
	}
	return orders, nil
}

func (p *Storage) GetBalance(ctx context.Context, uid uuid.UUID) (models.BalanceResponse, error) {
	var current int64
	var withdrawn int64
	var balance models.BalanceResponse

	ctx, cancel := context.WithTimeout(ctx, p.cfg.QueryTimeout)
	defer cancel()

	err := p.db.QueryRow(ctx, "SELECT current_balance, withdrawn FROM balances WHERE uid=$1 LIMIT 1", uid).Scan(&current, &withdrawn)
	if err == pgx.ErrNoRows {
		log.Printf("[ERROR] no balance for user %s %v", uid, err)
		return models.BalanceResponse{}, models.ErrBalanceNotFound
	}

	if err != nil {
		log.Printf("[ERROR] cannot get balance %v", err)
		return models.BalanceResponse{}, err
	}

	balance.Current = lib.RoundFloat(float64(current)/100.00, 2)
	balance.Withdrawn = lib.RoundFloat(float64(withdrawn)/100.00, 2)

	return balance, err
}

// TODO: проверить работу
// списание баллов в счет оплаты нового заказа
// здесь проблема с балансом
// если баланс меньше суммы списания, то списание не производится
func (p *Storage) SaveWithdraw(ctx context.Context, user models.User, order models.Order) (err error) {
	ctx, cancel := context.WithTimeout(ctx, p.cfg.QueryTimeout)
	defer cancel()

	tx, err := p.db.Begin(ctx)
	if err != nil {
		log.Printf("[ERROR] cannot begin tx %v", err)
		return err
	}
	defer tx.Rollback(ctx)

	bal, err := p.GetBalance(ctx, user.UID)
	if err != nil {
		log.Printf("[ERROR] cannot get balance %v", err)
		return models.ErrBalanceNotFound
	}

	if bal.Current < lib.RoundFloat(float64(order.Amount)/100.00, 2) {
		log.Printf("[ERROR] not enough balance %v", err)
		return models.ErrBalanceWrong
	}

	_, err = tx.Exec(
		ctx,
		"INSERT INTO withdrawals (order_id, uid, amount) VALUES ($1, $2, $3)",
		order.ID,
		user.UID,
		order.Amount,
	)
	if err != nil {
		log.Printf("[ERROR] cannot save withdrawal %v", err)
		return err
	}

	_, err = tx.Exec(
		ctx,
		"UPDATE balances SET current_balance=current_balance-$1, withdrawn=withdrawn+$1 WHERE uid=$2",
		order.Amount, user.UID,
	)
	if err != nil {
		log.Printf("[ERROR] cannot update balance for %s status %v", order.ID, err)
		return err
	}

	// err = tx.QueryRow(
	// 	ctx,
	// 	"UPDATE balances SET current_balance=current_balance-$1, withdrawn=withdrawn+$1 WHERE current_balance>=$1 AND uid=$2 RETURNING uid, current_balance, withdrawn",
	// 	order.Amount, user.UID,
	// ).Scan(&balance.UID, &balance.Current, &balance.Withdrawn)
	// if err != nil {
	// 	log.Printf("[ERROR] cannot update balance for %s status %v", order.ID, err)
	// 	return models.ErrBalanceWrong
	// }

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
		var amount int64
		order := models.WithdrawalsResponse{}
		err := rows.Scan(&order.Number, &amount, &order.ProcessedAt)
		if err != nil {
			log.Printf("[ERROR] cannot get order %v", err)
			continue
		}
		order.Accrual = lib.RoundFloat(float64(amount)/100.00, 2)
		orders = append(orders, order)
	}
	return orders, err
}

func (p *Storage) UpdateOrderStatus(ctx context.Context, orderNumber string, status models.AccrualStatus, amount int64) (models.OrderResponse, error) {
	var uid uuid.UUID

	order := models.OrderResponse{}

	tx, err := p.db.Begin(ctx)
	if err != nil {
		log.Printf("[ERROR] cannot begin tx %v", err)
		return order, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(
		ctx,
		"UPDATE orders SET status=$2, amount=$3 WHERE id=$1 RETURNING id, uid, amount, status, updated_at",
		orderNumber, status, amount,
	).Scan(&order.ID, &uid, &order.Amount, &order.Status, &order.UploadedAt)
	if err != nil {
		log.Printf("[ERROR] cannot update order %s status %v", orderNumber, err)
		return order, err
	}

	_, err = tx.Exec(
		ctx,
		"INSERT INTO balances (uid, current_balance, withdrawn) VALUES ($1, $2, $3) ON CONFLICT (uid) DO UPDATE SET current_balance = balances.current_balance + $2",
		uid, amount, 0,
	)
	if err != nil {
		log.Printf("[ERROR] cannot update balance for %s status %v", orderNumber, err)
		return order, err
	}

	return order, nil
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
