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
