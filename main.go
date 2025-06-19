package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/erigontech/assertoor/cmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM, syscall.SIGINT)

	runChan := make(chan bool)

	var execErr error
	go func() {
		execErr = cmd.Execute(ctx)

		close(runChan)
	}()

	select {
	case <-runChan:
		// normal exit
	case sig := <-signalChan:
		log.Printf("Caught signal: %v, shutdown gracefully...", sig)
		cancel()
		select {
		case <-runChan:
			// graceful shutdown completed
		case <-time.After(30 * time.Second):
			log.Println("Graceful shutdown timed out")
		}
	}

	if execErr != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
