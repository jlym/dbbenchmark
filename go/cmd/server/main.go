package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jlym/dbbenchmark/go/internal/postgres"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	action := ""
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	if action == "create" {
		pgServer := postgres.PGServer{
			Host:     "localhost",
			Password: "password",
			Port:     5432,
		}
		err := pgServer.CreateDB(ctx)
		if err != nil {
			log.Fatalf("%+v\n", err)
		}
	} else if action == "drop" {
		pgServer := postgres.PGServer{
			Host:     "localhost",
			Password: "password",
			Port:     5432,
		}
		err := pgServer.DropDB(ctx)
		if err != nil {
			log.Fatalf("%+v\n", err)
		}
	}
}
