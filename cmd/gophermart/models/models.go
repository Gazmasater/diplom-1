package models

type User struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}
