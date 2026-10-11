package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs-box/devctl/cmd"
)

func main() {
	// Establish a signal-aware root context so that an interrupt (Ctrl+C) or
	// termination signal cancels all in-flight work instead of leaving
	// subprocesses and network calls running.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd.Execute(ctx)
}
