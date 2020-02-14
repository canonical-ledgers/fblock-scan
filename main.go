package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/canonical-ledgers/fblock-scan/engine"
)

func main() {
	os.Exit(_main())
}
func _main() int {
	cfg := engine.NewConfig()
	parseFlags(&cfg)

	fmt.Println("fblock-scan: Factoid Block Transaction Scanner")
	fmt.Println(cfg)
	fmt.Println("Starting...")

	// Listen for an Interrupt and cancel everything if it occurs.
	ctx, cancel := context.WithCancel(context.Background())
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	go func() {
		<-sigint
		cancel()
	}()

	engineDone, err := cfg.Start(ctx)
	if err != nil {
		fmt.Println("Error: ", err)
		return 1
	}
	defer func() {
		<-engineDone
		fmt.Println("Engine stopped.")
	}()

	fmt.Println("Engine started.")

	defer func() {
		// Stop handling all signals so a force quit can occur with a
		// second sigint.
		signal.Reset()

		// Cause our sigint listener goroutine to call cancel().
		close(sigint)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("SIGINT: Shutting down...")
		return 0
	case <-engineDone:
	}
	return 1
}
