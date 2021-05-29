package main

import (
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/athos/trenchman/trenchman/repl"
)

const (
	COLOR_NONE   = "none"
	COLOR_ALWAYS = "always"
)

var opts struct {
	Host  string `name:"host" help:"host" default:"127.0.0.1"`
	Port  int    `name:"port" required:"true" help:"port"`
	Eval  string `name:"eval" short:"e" help:"eval"`
	Color string `name:"color" short:"c" enum:"always,none" default:"always" help:"color"`
}

func main() {
	kong.Parse(&opts)
	code := strings.TrimSpace(opts.Eval)
	oneshotEval := code != ""
	repl := repl.NewRepl(&repl.Opts{
		In:       os.Stdin,
		Out:      os.Stdout,
		Err:      os.Stderr,
		Printer:  repl.NewPrinter(opts.Color != COLOR_NONE),
		HidesNil: oneshotEval,
	}, func(ioHandler nrepl.IOHandler) *nrepl.Client {
		client, err := nrepl.NewClient(opts.Host, opts.Port, ioHandler)
		if err != nil {
			panic(err)
		}
		return client
	})
	defer repl.Close()

	if oneshotEval {
		repl.Eval(code)
		return
	}
	repl.StartWatchingInterruption()
	repl.Start()
}
