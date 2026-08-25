package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	agent "github.com/glueops/apiary-agent/cmd/agent"
)

// Stamped at build time with -ldflags "-X main.version=... -X main.commit=...".
// They are handed to the cobra package below rather than read from it, so the
// linker flags stay at their conventional main.* paths.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	agent.Version = version
	agent.Commit = commit
	agent.BuildDate = date

	os.Exit(run())
}

// run exists so the deferred stop() actually runs. log.Fatal calls os.Exit,
// which skips defers — leaving the signal handler registered on the way out.
// Harmless in a process that is ending, but it is the same shape as a defer that
// releases something that matters, and worth not teaching by example.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := agent.ExecuteContext(ctx); err != nil {
		log.Print(err)
		return 1
	}
	return 0
}
