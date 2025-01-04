package storage

import (
	"time"
)

type User struct {
	UserID    string
	UserName  string
	CreatedAt time.Time
	Role      string
}

type Post struct {
	PostID        string
	CreatorUserID string
	CreatedAt     time.Time
	Content       string
}
