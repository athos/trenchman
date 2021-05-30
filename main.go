package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/athos/trenchman/trenchman/repl"
	"github.com/mattn/go-isatty"
)

var version = "v0.0.0"

const (
	COLOR_NONE   = "none"
	COLOR_AUTO   = "auto"
	COLOR_ALWAYS = "always"
)

var opts struct {
	Host  string `name:"host" short:"H" help:"host" default:"127.0.0.1"`
	Port  int    `name:"port" short:"p" help:"port"`
	Eval  string `name:"eval" short:"e" help:"eval"`
	Color string `name:"color" short:"c" enum:"always,auto,none" default:"auto" help:"color"`
	Version bool `name:"version" short:"v" help:"Show version"`
}

func detectNreplPort(portFile string) (int, error) {
	content, err := os.ReadFile(portFile)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(string(content))
	if err != nil {
		return 0, err
	}
	return port, nil
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
	if opts.Version {
		fmt.Printf("Trenchman %s\n", version)
		os.Exit(0)
	}

	port := opts.Port
	if port == 0 {
		p, err := detectNreplPort(".nrepl-port")
		if err != nil {
			panic(fmt.Errorf("cannot read .nrepl-port (%w)", err))
		}
		port = p
	}
	code := strings.TrimSpace(opts.Eval)
	oneshotEval := code != ""
	repl := repl.NewRepl(&repl.Opts{
		In:       os.Stdin,
		Out:      os.Stdout,
		Err:      os.Stderr,
		Printer:  repl.NewPrinter(colorized(opts.Color)),
		HidesNil: oneshotEval,
	}, func(ioHandler nrepl.IOHandler) *nrepl.Client {
		client, err := nrepl.NewClient(opts.Host, port, ioHandler)
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
