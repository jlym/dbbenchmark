package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	p "github.com/jlym/dbbenchmark/go/internal/postgres"
	s "github.com/jlym/dbbenchmark/go/internal/server"
	"github.com/jlym/dbbenchmark/go/internal/util"
	"github.com/stretchr/testify/require"
)

func newTestEnv(ctx context.Context, t *testing.T) (*p.DBManager, *p.PGServer) {
	dbManager := p.NewDBManager(p.DevConnStringOptions)
	err := dbManager.InitDB(ctx)
	require.NoError(t, err)
	dbManager.TruncateTables(ctx)

	server, err := p.NewPGServer(ctx, p.DevConnStringOptions)
	require.NoError(t, err)

	return dbManager, server
}

func TestCreateUser(t *testing.T) {
	ctx, cancel := getTestContext()
	defer cancel()

	_, server := newTestEnv(ctx, t)
	defer server.Close()

	stubClock := util.NewStubClock()
	server.Clock = stubClock

	userName := gofakeit.Username()
	role := s.RoleViewer

	resp, err := server.CreateUser(ctx, &s.CreateUserRequest{
		UserName: userName,
		Role:     role,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.User)
	require.NotEmpty(t, resp.User.UserID)
	require.Equal(t, userName, resp.User.UserName)
	require.Equal(t, role, resp.User.Role)
	require.Equal(t, stubClock.NowUtc(), resp.User.CreatedAt)
}

func TestGetUser(t *testing.T) {
	ctx, cancel := getTestContext()
	defer cancel()

	_, server := newTestEnv(ctx, t)
	defer server.Close()

	createUserResp, err := server.CreateUser(ctx, &s.CreateUserRequest{
		UserName: gofakeit.Username(),
		Role:     s.RoleViewer,
	})
	require.NoError(t, err)
	createdUser := createUserResp.User

	getUserResp, err := server.GetUser(ctx, &s.GetUserRequest{
		CallerID: createdUser.UserID,
		UserID:   createdUser.UserID,
	})
	require.NoError(t, err)
	require.NotNil(t, getUserResp)
	require.Equal(t, createdUser, getUserResp.User)
}

func getTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}
