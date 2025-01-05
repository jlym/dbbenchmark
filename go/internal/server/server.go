package server

import "context"

type Server interface {
	CreateUser(ctx context.Context, request *CreateUserRequest) (*CreateUserResponse, error)
	GetUser(ctx context.Context, request *GetUserRequest) (*GetUserResponse, error)
	FollowUser(ctx context.Context, request *FollowUserRequest) (*FollowUserResponse, error)

	GetUserFeed(ctx context.Context, request *GetUserFeedRequest) (*GetUserFeedResponse, error)
	GetFollowedFeed(ctx context.Context, request *GetFollowedFeedRequest) (*GetFollowedFeedResponse, error)

	GetFollowed(ctx context.Context, request *GetFollowedRequest) (*GetFollowedResponse, error)

	CreatePost(ctx context.Context, request *CreatePostRequest) (*CreatePostResponse, error)
	GetPost(ctx context.Context, request *GetPostRequest) (*GetPostResponse, error)
	LikePost(ctx context.Context, request *LikePostRequest) (*LikePostResponse, error)
}
