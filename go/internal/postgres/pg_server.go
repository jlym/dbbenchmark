package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"

	e "errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	s "github.com/jlym/dbbenchmark/go/internal/server"
	"github.com/jlym/dbbenchmark/go/internal/util"
)

const (
	dbFeed     = "feeddb"
	dbPostgres = "postgres"
)

type PGServer struct {
	DBPool *pgxpool.Pool
	Clock  util.Clock
}

// Enforce that PGServer implements s.Server interface.
// var _ s.Server = &PGServer{}

func NewPGServer(parentCtx context.Context, connOptions *ConnStringOptions) (*PGServer, error) {
	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, connOptions.GetConnString(dbFeed))
	if err != nil {
		return nil, errors.Wrapf(err, "creating connection pool failed, connString=\"%s\"", connOptions.GetDebugConnString(dbFeed))
	}

	return &PGServer{
		DBPool: dbPool,
		Clock:  util.NewRealClock(),
	}, nil
}

func (p *PGServer) Close() {
	p.DBPool.Close()
}

func (p *PGServer) CreateUser(
	parentCtx context.Context,
	request *s.CreateUserRequest) (*s.CreateUserResponse, error) {

	if request.UserName == "" {
		return nil, errors.New("request.UserName was empty")
	} else if request.Role == "" {
		return nil, errors.New("request.Role was empty")
	}

	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()
	row := p.DBPool.QueryRow(ctx, `
		INSERT INTO users (user_id, user_name, created_at, role)
		VALUES (gen_random_uuid(), $1, $2, $3)
		RETURNING user_id, user_name, created_at, role
	`, request.UserName, p.Clock.NowUtc(), request.Role)

	var userID string
	var userName string
	var createdAt time.Time
	var role s.Role
	err := row.Scan(&userID, &userName, &createdAt, &role)
	if err != nil {
		return nil, errors.Wrap(err, "creating user failed")
	}

	return &s.CreateUserResponse{
		User: &s.User{
			UserID:           userID,
			UserName:         userName,
			Role:             role,
			CreatedAt:        createdAt.UTC(),
			FollowedByCaller: false,
		},
	}, nil
}

func (p *PGServer) GetUser(parentCtx context.Context, request *s.GetUserRequest) (*s.GetUserResponse, error) {
	if request.CallerID == "" {
		return nil, errors.New("request.CallerID was empty")
	} else if request.UserID == "" {
		return nil, errors.New("request.UserID was empty")
	}
	callerID, userID := request.CallerID, request.UserID

	// Query for the user's information.
	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	row := p.DBPool.QueryRow(ctx, `
		SELECT user_name, created_at, role
		FROM users
		WHERE user_id = $1
		LIMIT 1;
	`, userID)

	var userName string
	var createdAt time.Time
	var role s.Role
	err := row.Scan(&userName, &createdAt, &role)
	if err == sql.ErrNoRows {
		return &s.GetUserResponse{}, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "creating user failed")
	}

	// Check if caller follows the user.
	followedByCaller := false
	if request.CallerID != request.UserID {
		ctx, cancel = getQueryContext(parentCtx)
		defer cancel()

		row = p.DBPool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT source_id, target_id
				FROM follows
				WHERE source_id = $1 AND target_id = $2
				LIMIT 1
			);
		`, callerID, userID)
		err = row.Scan(&followedByCaller)
		if err != nil {
			return nil, errors.Wrap(err, "checking for follow failed")
		}
	}

	return &s.GetUserResponse{
		User: &s.User{
			UserID:           userID,
			UserName:         userName,
			Role:             role,
			CreatedAt:        createdAt.UTC(),
			FollowedByCaller: followedByCaller,
		},
	}, nil
}

func (p *PGServer) FollowUser(
	ctx context.Context, request *s.FollowUserRequest) (*s.FollowUserResponse, error) {

	if request.CallerID == "" {
		return nil, errors.New("request.CallerID was empty")
	} else if request.TargetUserID == "" {
		return nil, errors.New("request.TargetUserID was empty")
	}
	callerID, targetUserID := request.CallerID, request.TargetUserID

	innerCtx, cancel := getQueryContext(ctx)
	defer cancel()
	tx, err := p.DBPool.Begin(innerCtx)
	if err != nil {
		return nil, errors.Wrap(err, "starting transaction failed")
	}

	err = p.assertUserExist(ctx, tx, callerID)
	if err != nil {
		return nil, p.rollbackDueToError(ctx, tx, err)
	}
	err = p.assertUserExist(ctx, tx, targetUserID)
	if err != nil {
		return nil, p.rollbackDueToError(ctx, tx, err)
	}

	innerCtx, cancel = getQueryContext(ctx)
	defer cancel()
	_, err = tx.Exec(innerCtx, `
		INSERT INTO follows (source_id, target_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`, callerID, targetUserID, p.Clock.NowUtc())
	if err != nil {
		return nil, p.rollbackDueToError(ctx, tx, err)
	}

	innerCtx, cancel = getQueryContext(ctx)
	defer cancel()
	err = tx.Commit(innerCtx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &s.FollowUserResponse{}, nil
}

func (p *PGServer) CreatePost(
	ctx context.Context, request *s.CreatePostRequest) (*s.CreatePostResponse, error) {

	if request.CallerID == "" {
		return nil, errors.New("request.CallerID was empty")
	} else if request.Content == "" {
		return nil, errors.New("request.Content was empty")
	}
	callerID, content := request.CallerID, request.Content

	innerCtx, cancel := getQueryContext(ctx)
	defer cancel()

	row := p.DBPool.QueryRow(innerCtx, `
		INSERT INTO posts (post_id, owner_id, created_at, content)
		VALUES (gen_random_uuid(), $1, $2, $3)
		RETURNING post_id, owner_id, created_at, content
	`, callerID, p.Clock.NowUtc(), content)

	var postID string
	var ownerID string
	var createdAt time.Time
	err := row.Scan(&postID, &ownerID, &createdAt, &content)
	if err != nil {
		return nil, errors.Wrap(err, "creating post failed")
	}

	return &s.CreatePostResponse{
		CallerID: callerID,
		Post: &s.Post{
			PostID:        postID,
			OwnerID:       ownerID,
			Content:       content,
			CreatedAt:     createdAt.UTC(),
			LikeCount:     0,
			LikedByCaller: false,
		},
	}, nil
}

func (p *PGServer) GetPost(
	ctx context.Context, request *s.GetPostRequest) (*s.GetPostResponse, error) {

	if request.CallerID == "" {
		return nil, errors.New("request.CallerID was empty")
	} else if request.PostID == "" {
		return nil, errors.New("request.PostID was empty")
	}
	callerID, postID := request.CallerID, request.PostID

	// Query for post.
	innerCtx, cancel := getQueryContext(ctx)
	defer cancel()

	row := p.DBPool.QueryRow(innerCtx, `
		SELECT owner_id, created_at, content
		FROM posts
		WHERE post_id = $1
		LIMIT 1;
	`, postID)

	var ownerID string
	var createdAt time.Time
	var content string
	err := row.Scan(&ownerID, &createdAt, &content)
	if err == sql.ErrNoRows {
		return &s.GetPostResponse{}, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "querying for post failed")
	}

	// Query for like count.
	innerCtx, cancel = getQueryContext(ctx)
	defer cancel()

	row = p.DBPool.QueryRow(innerCtx, `
		SELECT COUNT(*)
		FROM likes
		WHERE post_id = $1;
	`, postID)
	var likeCount int
	err = row.Scan(&likeCount)
	if err != nil {
		return nil, errors.Wrap(err, "querying for like count failed")
	}

	// Query to see if post is liked by caller.
	innerCtx, cancel = getQueryContext(ctx)
	defer cancel()

	row = p.DBPool.QueryRow(innerCtx, `
		SELECT EXISTS (
			SELECT post_id, user_id
			FROM likes
			WHERE post_id = $1 AND user_id = $2
		);
	`, postID, callerID)
	var likedByCaller bool
	err = row.Scan(&likedByCaller)
	if err != nil {
		return nil, errors.Wrap(err, "querying to see if post is liked by caller failed")
	}

	return &s.GetPostResponse{
		Post: &s.Post{
			PostID:        postID,
			OwnerID:       ownerID,
			Content:       content,
			CreatedAt:     createdAt.UTC(),
			LikeCount:     likeCount,
			LikedByCaller: likedByCaller,
		},
	}, nil
}

func (p *PGServer) LikePost(
	ctx context.Context, request *s.LikePostRequest) (*s.LikePostResponse, error) {

	if request.CallerID == "" {
		return nil, errors.New("request.CallerID was empty")
	} else if request.PostID == "" {
		return nil, errors.New("request.PostID was empty")
	}
	callerID, postID := request.CallerID, request.PostID

	innerCtx, cancel := getQueryContext(ctx)
	defer cancel()

	_, err := p.DBPool.Exec(innerCtx, `
		INSERT INTO likes (post_id, user_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING;
	`, postID, callerID, p.Clock.NowUtc())
	if err != nil {
		return nil, errors.Wrap(err, "liking post failed")
	}

	getPostResp, err := p.GetPost(ctx, &s.GetPostRequest{
		CallerID: callerID,
		PostID:   postID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting updated post failed")
	}

	return &s.LikePostResponse{
		Post: getPostResp.Post,
	}, nil
}

func (p *PGServer) assertUserExist(ctx context.Context, tx pgx.Tx, userID string) error {
	innerCtx, cancel := getQueryContext(ctx)
	defer cancel()

	row := tx.QueryRow(innerCtx, `
		SELECT EXISTS (
			SELECT user_id FROM users WHERE user_id = $1 LIMIT 1
		);
	`, userID)
	var exists bool
	err := row.Scan(&exists)
	if err != nil {
		return errors.WithStack(err)
	} else if !exists {
		return fmt.Errorf("given user does not exist, userID=\"%s\"", userID)
	}

	return nil
}

func getQueryContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parentCtx, 10*time.Second)
}

func (p *PGServer) rollbackDueToError(ctx context.Context, tx pgx.Tx, err error) error {
	innerCtx, cancel := getQueryContext(ctx)
	defer cancel()

	txErr := tx.Rollback(innerCtx)
	if txErr != nil {
		return e.Join(err, txErr)
	}

	return err
}
