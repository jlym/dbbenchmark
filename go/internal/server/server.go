package server

type Server interface {
	CreateUser(request *CreateUserRequest) (*CreateUserResponse, error)
	GetUser(request *GetUserRequest) (*GetUserResponse, error)
	FollowUser(request *FollowUserRequest) (*FollowUserResponse, error)

	GetUserFeed(request *GetUserFeedRequest) (*GetUserFeedResponse, error)
	GetFollowedFeed(request *GetFollowedFeedRequest) (*GetFollowedFeedResponse, error)

	GetFollowed(request *GetFollowedRequest) (*GetFollowedResponse, error)

	CreatePost(request *CreatePostRequest) (*CreatePostResponse, error)
	GetPost(request *GetPostRequest) (*GetPostResponse, error)
	LikePost(request *LikePostRequest) (*LikePostResponse, error)
}
