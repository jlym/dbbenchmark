package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/pkg/errors"

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

	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	row := p.DBPool.QueryRow(ctx, `
		SELECT user_id, user_name, created_at, role
		FROM users
		WHERE user_id = $1
		LIMIT 1;
	`, request.UserID)

	var userID string
	var userName string
	var createdAt time.Time
	var role s.Role
	found := true
	err := row.Scan(&userID, &userName, &createdAt, &role)
	if err == sql.ErrNoRows {
		found = false
	} else if err != nil {
		return nil, errors.Wrap(err, "creating user failed")
	}

	// TODO: Implement checking if caller follows the user.

	var user *s.User
	if found {
		user = &s.User{
			UserID:           userID,
			UserName:         userName,
			Role:             role,
			CreatedAt:        createdAt.UTC(),
			FollowedByCaller: false,
		}
	}
	return &s.GetUserResponse{
		User: user,
	}, nil
}

func getQueryContext(parentCtx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parentCtx, 10*time.Second)
}
