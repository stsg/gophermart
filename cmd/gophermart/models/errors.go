package models

import (
	"fmt"
)

var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserExists        = fmt.Errorf("user exists")
	ErrUserUnauthorized  = fmt.Errorf("user unauthorized")
	ErrUserWrong         = fmt.Errorf("user wrong")
	ErrUserWrongPassword = fmt.Errorf("user password wrong")

	ErrOrderNotFound           = fmt.Errorf("order not found")
	ErrOrderExists             = fmt.Errorf("order exists")
	ErrOrderBelongsAnotherUser = fmt.Errorf("order belongs to another user")
	ErrOrderWrong              = fmt.Errorf("order wrong")

	ErrWithdrawalNotFound = fmt.Errorf("withdrawal not found")
	ErrWithdrawalExists   = fmt.Errorf("withdrawal exists")
	ErrWithdrawalWrong    = fmt.Errorf("withdrawal wrong")

	ErrBalanceNotFound = fmt.Errorf("balance not found")
	ErrBalanceExists   = fmt.Errorf("balance exists")
	ErrBalanceWrong    = fmt.Errorf("balance wrong")
)
