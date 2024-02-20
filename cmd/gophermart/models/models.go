package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID     uuid.UUID `json:"uuid,omitempty" db:"uuid"`
	Login    string    `json:"login,omitempty" db:"login"`
	PHash    string    `json:"p_hash,omitempty" db:"p_hash"`
	JWTToken string    `json:"jwt_token,omitempty" db:"jwt_token"`
}

var (
	ErrUserNotFound = fmt.Errorf("user not found")
	ErrUserExists   = fmt.Errorf("user exists")
	ErrUserWrong    = fmt.Errorf("user wrong")
)

type AccrualStatus string

const (
	AccrualStatusNew        AccrualStatus = "NEW"
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualStatus = "PROCESSED"
	AccrualStatusInvalid    AccrualStatus = "INVALID"
)

type Order struct {
	ID            string        `json:"number" db:"id"`
	UUID          uuid.UUID     `json:"uuid" db:"uuid"`
	Accrual       int64         `json:"accrual" db:"accrual"`
	AccrualStatus AccrualStatus `json:"status" db:"accrual_status"`
	UploadedAt    time.Time     `json:"uploaded_at" db:"uploaded_at"`
}

var (
	ErrOrderNotFound = fmt.Errorf("order not found")
	ErrOrderExists   = fmt.Errorf("order exists")
	ErrOrderWrong    = fmt.Errorf("order wrong")
)

type Withdrawal struct {
	OrderID     string    `json:"order" db:"order_id"`
	UUID        uuid.UUID `json:"-" db:"uid"`
	Amount      int64     `json:"sum" db:"amount"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}

var (
	ErrWithdrawalNotFound = fmt.Errorf("withdrawal not found")
	ErrWithdrawalExists   = fmt.Errorf("withdrawal exists")
	ErrWithdrawalWrong    = fmt.Errorf("withdrawal wrong")
)
