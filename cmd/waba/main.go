// Command waba is a command-line client for Meta's WhatsApp Cloud API.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jjuanrivvera/waba-cli/commands"
)

func main() {
	// Ctrl-C must cancel work in flight, not just kill the process between requests: an
	// --all walk over a large template list, a retry backoff and a media download all check
	// this context. Every call site threads cmd.Context() from here.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// User aliases are expanded before cobra sees the arguments — there is no cobra hook
	// early enough, and rewriting args later fights its flag parsing. Built-ins always win,
	// so an alias can never shadow a real command.
	root := commands.NewRootCmd()
	os.Args = append(os.Args[:1], commands.ExpandAlias(os.Args[1:], commands.BuiltinNames(root))...)

	os.Exit(commands.Execute(ctx))
}
