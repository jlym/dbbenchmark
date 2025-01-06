package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jlym/dbbenchmark/go/internal/postgres"
)

func main() {
	action := ""
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	err := run(action)
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
}

func run(action string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbManager := postgres.DBManager{
		Options: postgres.DevConnStringOptions,
	}

	if action == "create" {
		err := dbManager.InitDB(ctx)
		if err != nil {
			return err
		}
	} else if action == "drop" {
		err := dbManager.DropDB(ctx)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported action: \"%s\"", action)
	}

	return nil
}
