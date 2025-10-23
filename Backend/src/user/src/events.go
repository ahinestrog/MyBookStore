package main

type UserCreated struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

type UserUpdated struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}
