package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	UID      uuid.UUID `json:"uuid,omitempty" db:"uuid"`
	Login    string    `json:"login,omitempty" db:"login"`
	PHash    string    `json:"p_hash,omitempty" db:"p_hash"`
	JWTToken string    `json:"jwt_token,omitempty" db:"jwt_token"`
}

var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserExists        = fmt.Errorf("user exists")
	ErrUserUnauthorized  = fmt.Errorf("user unauthorized")
	ErrUserWrong         = fmt.Errorf("user wrong")
	ErrUserWrongPassword = fmt.Errorf("user password wrong")
)

type UserRegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Order struct {
	ID            string        `json:"number" db:"id"`
	UID           uuid.UUID     `json:"uuid" db:"uuid"`
	Amount        int64         `json:"accrual" db:"accrual"`
	AccrualStatus AccrualStatus `json:"status" db:"accrual_status"`
	UploadedAt    time.Time     `json:"uploaded_at" db:"uploaded_at"`
}

var (
	ErrOrderNotFound = fmt.Errorf("order not found")
	ErrOrderExists   = fmt.Errorf("order exists")
	ErrOrderWrong    = fmt.Errorf("order wrong")
)

type OrderResponse struct {
	ID         string    `json:"number"`
	Username   string    `json:"username"`
	Status     string    `json:"status"`
	Amount     int64     `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type AccrualStatus string

const (
	AccrualStatusNew        AccrualStatus = "NEW"
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualStatus = "PROCESSED"
	AccrualStatusInvalid    AccrualStatus = "INVALID"
)

type Accruals struct {
	OrderID     string    `json:"order" db:"order_id"`
	UID         uuid.UUID `json:"uuid" db:"uid"`
	Amount      int64     `json:"sum" db:"amount"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}

var (
	ErrWithdrawalNotFound = fmt.Errorf("withdrawal not found")
	ErrWithdrawalExists   = fmt.Errorf("withdrawal exists")
	ErrWithdrawalWrong    = fmt.Errorf("withdrawal wrong")
)

type AccrualResponse struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
}

type Balance struct {
	UID       uuid.UUID `json:"uuid" db:"uid"`
	Current   int       `json:"current" db:"current_balance"`
	Withdrawn int       `json:"withdrawn" db:"withdrawn"`
	// UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
}

type BalanceResponse struct {
	Current    int       `json:"current"`
	Withdrawn  int       `json:"withdrawn"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type WithdrawRequest struct {
	Number  string `json:"order"`
	Accrual int64  `json:"sum"`
}

type WithdrawResponse struct {
	Number      string    `json:"order"`
	Accrual     int       `json:"sum,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}
