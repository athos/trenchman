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

var args struct {
	Host     string `name:"host" short:"H" help:"host" default:"127.0.0.1"`
	Port     int    `name:"port" short:"p" help:"port"`
	Eval     string `name:"eval" short:"e" help:"eval"`
	Color    string `name:"color" short:"c" enum:"always,auto,none" default:"auto" help:"color"`
	Version  bool   `name:"version" short:"v" help:"Show version"`
	Filename string `arg:"true" optional:"true"`
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
	kong.Parse(&args)
	if args.Version {
		fmt.Printf("Trenchman %s\n", version)
		os.Exit(0)
	}

	port := args.Port
	if port == 0 {
		p, err := detectNreplPort(".nrepl-port")
		if err != nil {
			panic(fmt.Errorf("cannot read .nrepl-port (%w)", err))
		}
		port = p
	}
	filename := strings.TrimSpace(args.Filename)
	code := strings.TrimSpace(args.Eval)
	repl := repl.NewRepl(&repl.Opts{
		In:       os.Stdin,
		Out:      os.Stdout,
		Err:      os.Stderr,
		Printer:  repl.NewPrinter(colorized(args.Color)),
		HidesNil: filename != "" || code != "",
	}, func(ioHandler nrepl.IOHandler) *nrepl.Client {
		client, err := nrepl.NewClient(args.Host, port, ioHandler)
		if err != nil {
			panic(err)
		}
		return client
	})
	defer repl.Close()

	if filename != "" {
		repl.Load(filename)
		return
	}
	if code != "" {
		repl.Eval(code)
		return
	}
	repl.StartWatchingInterruption()
	repl.Start()
}
