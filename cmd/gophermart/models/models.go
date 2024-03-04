package models

import (
	"time"

	"github.com/google/uuid"
)

type AccrualStatus string

const (
	AccrualStatusNew        AccrualStatus = "NEW"
	AccrualStatusProcessing AccrualStatus = "PROCESSING"
	AccrualStatusProcessed  AccrualStatus = "PROCESSED"
	AccrualStatusInvalid    AccrualStatus = "INVALID"
)

type User struct {
	UID      uuid.UUID `json:"uuid,omitempty" db:"uuid"`
	Login    string    `json:"login,omitempty" db:"login"`
	PHash    string    `json:"p_hash,omitempty" db:"p_hash"`
	JWTToken string    `json:"jwt_token,omitempty" db:"jwt_token"`
}

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

type OrderResponse struct {
	ID         string    `json:"number"`
	Status     string    `json:"status"`
	Amount     float64   `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type Accrual struct {
	OrderID string    `json:"order" db:"order_id"`
	UID     uuid.UUID `json:"uuid" db:"uid"`
	Amount  int64     `json:"sum" db:"amount"`
	// ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}

type AccrualResponse struct {
	Order   string        `json:"order"`
	Status  AccrualStatus `json:"status"`
	Accrual float64       `json:"accrual"`
}

type Balance struct {
	UID       uuid.UUID `json:"uuid" db:"uid"`
	Current   int       `json:"current" db:"current_balance"`
	Withdrawn int       `json:"withdrawn" db:"withdrawn"`
	// UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
}

type BalanceResponse struct {
	Current    float64   `json:"current"`
	Withdrawn  float64   `json:"withdrawn"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type WithdrawRequest struct {
	Number  string  `json:"order"`
	Accrual float64 `json:"sum"`
}

type WithdrawResponse struct {
	Number      string    `json:"order"`
	Accrual     float64   `json:"sum,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}

type WithdrawalsResponse struct {
	Number      string    `json:"order"`
	Accrual     float64   `json:"sum,omitempty"`
	ProcessedAt time.Time `json:"processed_at"`
}
