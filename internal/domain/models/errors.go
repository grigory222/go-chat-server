package models

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrChatNotFound       = errors.New("chat not found")
	ErrAccessDenied       = errors.New("access denied")
)
