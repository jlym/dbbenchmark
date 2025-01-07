package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

type DBManager struct {
	Options *ConnStringOptions
}

func NewDBManager(options *ConnStringOptions) *DBManager {
	return &DBManager{
		Options: options,
	}
}

func (d *DBManager) InitDB(parentCtx context.Context) error {
	postgresConn, err := d.openConn(parentCtx, dbPostgres)
	if err != nil {
		return err
	}
	defer d.closeConn(parentCtx, postgresConn)

	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	feedsDBExists := false
	row := postgresConn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT datname
			FROM pg_catalog.pg_database
			WHERE datname = $1
			LIMIT 1
		);
	`, dbFeed)
	err = row.Scan(&feedsDBExists)
	if err != nil {
		return errors.Wrap(err, "checking if feeds database exists failed")
	}

	if feedsDBExists {
		return nil
	}

	_, err = postgresConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s;", dbFeed))
	if err != nil {
		return errors.Wrap(err, "creating feeds db failed")
	}

	feedConn, err := d.openConn(parentCtx, dbFeed)
	if err != nil {
		return err
	}
	defer d.closeConn(parentCtx, feedConn)

	ctx, cancel = getQueryContext(parentCtx)
	defer cancel()

	_, err = feedConn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			user_id UUID PRIMARY KEY,
			user_name TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			ROLE TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS follows (
			source_id UUID NOT NULL,
			target_id UUID NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			PRIMARY KEY(source_id, target_id)
		);

		CREATE TABLE IF NOT EXISTS posts (
			post_id UUID PRIMARY KEY,
			owner_id UUID NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			content TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS likes (
			post_id UUID NOT NULL,
			user_id UUID NOT NULL,
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

func (d *DBManager) DropDB(parentCtx context.Context) error {
	conn, err := d.openConn(parentCtx, dbPostgres)
	if err != nil {
		return err
	}
	defer d.closeConn(parentCtx, conn)

	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	_, err = conn.Exec(ctx, fmt.Sprintf("DROP DATABASE %s;", dbFeed))
	if err != nil {
		return errors.Wrap(err, "dropping db failed")
	}

	return nil
}

func (d *DBManager) TruncateTables(parentCtx context.Context) error {
	conn, err := d.openConn(parentCtx, dbFeed)
	if err != nil {
		return err
	}
	defer d.closeConn(parentCtx, conn)

	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	_, err = conn.Exec(ctx, `
		TRUNCATE users, follows, posts, likes;
	`)
	if err != nil {
		return errors.Wrap(err, "clearing feed db failed")
	}

	return nil
}

func (d *DBManager) openConn(ctx context.Context, dbName string) (*pgx.Conn, error) {
	connString := d.Options.GetConnString(dbName)
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, errors.Wrapf(err, "opening connection failed, connString=\"%s\"", connString)
	}

	return conn, nil
}

func (d *DBManager) closeConn(parentCtx context.Context, conn *pgx.Conn) {
	ctx, cancel := getQueryContext(parentCtx)
	defer cancel()

	err := conn.Close(ctx)
	if err != nil {
		log.Printf("closing connection failed: %+v\n", err)
	}
}
