package models

import "time"

type Chat struct {
	ID        int64
	Name      string
	Type      string
	CreatedAt time.Time
}

type Message struct {
	ID        int64
	ChatID    int64
	UserID    int64
	UserName  string
	Text      string
	CreatedAt time.Time
}
