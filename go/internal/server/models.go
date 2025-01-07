package server

import (
	"time"
)

type Role string

const (
	RoleLargeCreator Role = "LargeCreator"
	RoleSmallCreator Role = "SmallCreator"
	RoleViewer       Role = "Viewer"
)

type CreateUserRequest struct {
	UserName string
	Role     Role
}

type CreateUserResponse struct {
	User *User
}

type GetUserFeedRequest struct {
	CallerID string
	OwnerID  string
	Limit    int
	Cursor   string
}

type GetUserFeedResponse struct {
	CallerID string
	OwnerID  string
	Posts    []*Post
	Limit    int
	Cursor   string
}

type GetFollowedFeedRequest struct {
	CallerID string
	Limit    int
	Cursor   string
}

type GetFollowedFeedResponse struct {
	CallerID string
	OwnerID  string
	Posts    []*Post
	Limit    int
	Cursor   string
}

type GetFollowedRequest struct {
	CallerID string
	Limit    int
	Cursor   string
}

type GetFollowedResponse struct {
	CallerID string
	OwnerID  string
	User     []*User
	Limit    int
	Cursor   string
}

type GetUserRequest struct {
	CallerID string
	UserID   string
}

type GetUserResponse struct {
	User *User
}

type FollowUserRequest struct {
	CallerID     string
	TargetUserID string
}

type FollowUserResponse struct{}

type CreatePostRequest struct {
	CallerID string
	Content  string
}

type CreatePostResponse struct {
	CallerID string
	Post     *Post
}

type LikePostRequest struct {
	CallerID string
	PostID   string
}

type LikePostResponse struct {
	Post *Post
}

type GetPostRequest struct {
	CallerID string
	PostID   string
}

type GetPostResponse struct {
	Post *Post
}

type Post struct {
	PostID        string
	OwnerID       string
	Content       string
	CreatedAt     time.Time
	LikeCount     int
	LikedByCaller bool
}

type User struct {
	UserID           string
	UserName         string
	Role             Role
	CreatedAt        time.Time
	FollowedByCaller bool
}
