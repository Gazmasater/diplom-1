package models

import (
	"database/sql"
	"time"
)

type User struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	CreatedAt string `json:"created_at"`
}

type Order struct {
	ID            int          `json:"id"`
	OrderNumber   string       `json:"order_number"`
	Status        string       `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
	Accrual       float64      `json:"accrual"`
	Deduction     float64      `json:"deduction"`
	DeductionTime sql.NullTime `json:"deduction_time"`
	UserEmail     string       `json:"user_email"`
}

type Token struct {
	ID             int       `json:"id"`
	UserEmail      string    `json:"user_email"`
	Token          string    `json:"token"`
	CreatedAt      time.Time `json:"created_at"`
	ExpirationTime time.Time `json:"expiration_time"`
}
