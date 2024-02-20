// service, that implements business logic of gophermart

package service

import (
	"context"

	"github.com/stsg/gophermart/cmd/gophermart/models"
)

type Service interface {
	Registration(ctx context.Context, u models.User) (string, error)
	Login(ctx context.Context, u models.User) (string, error)
	PostOrders(ctx context.Context, u models.User) error
	GetOrders(ctx context.Context, u models.User) error
	GetBalance(ctx context.Context, u models.User) (int64, error)
	Withdraw(ctx context.Context, u models.User) error
	GetWithdrawals(ctx context.Context, u models.User) error
}
