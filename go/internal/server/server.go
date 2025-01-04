package server

type Server interface {
	CreateUser(request *CreateUserRequest) (*CreateUserResponse, error)

	GetUserFeed(request *GetUserFeedRequest) (*GetUserFeedResponse, error)
	GetFollowedFeed(request *GetFollowedFeedRequest) (*GetFollowedFeedResponse, error)

	FollowUser(request *FollowUserRequest) (*FollowUserResponse, error)
	GetFollowed(request *GetFollowedRequest) (*GetFollowedResponse, error)

	CreatePost(request *CreatePostRequest) (*CreatePostResponse, error)
	LikePost(request *LikePostRequest) (*LikePostResponse, error)
}
