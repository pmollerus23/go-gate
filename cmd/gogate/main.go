package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/pmollerus23/gogate/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
