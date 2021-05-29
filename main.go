package main

import (
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/athos/trenchman/trenchman/repl"
	"github.com/mattn/go-isatty"
)

const (
	COLOR_NONE   = "none"
	COLOR_AUTO   = "auto"
	COLOR_ALWAYS = "always"
)

var opts struct {
	Host  string `name:"host" short:"H" help:"host" default:"127.0.0.1"`
	Port  int    `name:"port" short:"p" required:"true" help:"port"`
	Eval  string `name:"eval" short:"e" help:"eval"`
	Color string `name:"color" short:"c" enum:"always,auto,none" default:"auto" help:"color"`
}

func colorized(color string) bool {
	switch color {
	case COLOR_NONE:
		return false
	case COLOR_ALWAYS:
		return true
	case COLOR_AUTO:
		if isatty.IsTerminal(os.Stdout.Fd()) ||
			isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			return true
		}
	}
	return false
}

func main() {
	kong.Parse(&opts)
	code := strings.TrimSpace(opts.Eval)
	oneshotEval := code != ""
	repl := repl.NewRepl(&repl.Opts{
		In:       os.Stdin,
		Out:      os.Stdout,
		Err:      os.Stderr,
		Printer:  repl.NewPrinter(colorized(opts.Color)),
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
