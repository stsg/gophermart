// service, that implements business logic of gophermart

package service

import "context"

type Withdrawal struct {
	ID     int
	Login  string
	Amount int64
}
type Service interface {
	Registration(ctx context.Context, login, password string) (string, error)
	Login(ctx context.Context, login, password string) (string, error)
	Withdraw(ctx context.Context, login, password string, amount int64) error
	Withdrawals(ctx context.Context, login, password string) ([]Withdrawal, error)
	Balance(ctx context.Context, login, password string) (int64, error)
}
