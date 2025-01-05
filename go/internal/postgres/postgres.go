package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"

	"github.com/jackc/pgx/v5"
)

const (
	dbFeed     = "feeddb"
	dbPostgres = "postgres"
)

type PGServer struct {
	Host     string
	Password string
	Port     int
}

// var _ s.Server = &PGServer{}

func (p *PGServer) CreateDB(ctx context.Context) error {
	postgesConn, closePostgresConn, err := p.openConn(ctx, dbPostgres)
	if err != nil {
		return err
	}
	defer closePostgresConn()

	_, err = postgesConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s;", dbFeed))
	if err != nil {
		return errors.Wrap(err, "creating db failed")
	}

	feedConn, closeFeedConn, err := p.openConn(ctx, dbFeed)
	if err != nil {
		return err
	}
	defer closeFeedConn()

	_, err = feedConn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			user_name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			ROLE TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS follows (
			source_id TEXT NOT NULL,
			target_id TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY(source_id, target_id)
		);

		CREATE TABLE IF NOT EXISTS posts (
			post_id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			content TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS likes (
			post_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY(post_id, user_id)
		);

		CREATE INDEX IF NOT EXISTS likes_post_id_idx ON likes (post_id);
	`)
	if err != nil {
		return errors.Wrap(err, "initializing feed db failed")
	}

	return nil
}

func (p *PGServer) DropDB(ctx context.Context) error {
	postgesConn, closePostgresConn, err := p.openConn(ctx, dbPostgres)
	if err != nil {
		return err
	}
	defer closePostgresConn()

	_, err = postgesConn.Exec(ctx, fmt.Sprintf("DROP DATABASE %s;", dbFeed))
	if err != nil {
		return errors.Wrap(err, "dropping db failed")
	}

	return nil
}

func (p *PGServer) openConn(ctx context.Context, dbName string) (*pgx.Conn, func(), error) {
	connString := p.getConnString(dbName)
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "opening connection failed, connString=\"%s\"", connString)
	}

	closeConn := func() {
		err = conn.Close(ctx)
		if err != nil {
			log.Printf("closing connection failed: %+v\n", err)
		}
	}

	return conn, closeConn, nil
}

func (p *PGServer) getConnString(dbName string) string {
	return fmt.Sprintf(
		"postgres://%s:%d/%s?user=postgres&password=%s",
		p.Host,
		p.Port,
		dbName,
		p.Password,
	)
}
