package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/albedosehen/rwwwrse/internal/di"
	"github.com/albedosehen/rwwwrse/internal/observability"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	app, err := di.InitializeApplication(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	if err := app.Start(ctx); err != nil {
		app.Logger.Error(ctx, err, "Failed to start application")
		os.Exit(1)
	}

	app.Logger.Info(ctx, "rwwwrse is running",
		observability.String("pid", fmt.Sprintf("%d", os.Getpid())),
	)

	<-sigChan
	app.Logger.Info(ctx, "Shutdown signal received, stopping server...")
	cancel()

	if err := app.Stop(ctx); err != nil {
		app.Logger.Error(ctx, err, "Error during shutdown")
		os.Exit(1)
	}

	app.Logger.Info(ctx, "rwwwrse stopped successfully")
}
