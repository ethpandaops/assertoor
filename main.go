package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethpandaops/assertoor/cmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	cancelSig := make(chan os.Signal, 1)
	signal.Notify(cancelSig, syscall.SIGTERM, syscall.SIGINT)

	runChan := make(chan bool)
	go func() {
		cmd.Execute(ctx)
		close(runChan)
	}()

	sig := <-cancelSig
	log.Printf("Caught signal: %v, shutdown gracefully...", sig)

	cancel()

	select {
	case <-runChan:
	case <-time.After(5 * time.Second):
		log.Println("Graceful shutdown timed out")
	}

	os.Exit(0)
}
