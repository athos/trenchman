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
	Host        string `name:"host" help:"Connect to the specified host." default:"127.0.0.1"`
	Port        int    `name:"port" short:"p" placeholder:"<port>" help:"Connect to the specified port."`
	Eval        string `name:"eval" short:"e" group:"Evaluation" placeholder:"<expr>" help:"Evaluate an expression."`
	File        string `name:"file" short:"f" group:"Evaluation" placeholder:"<path>" help:"Evaluate a file."`
	MainNS      string `name:"main" short:"m" group:"Evaluation" placeholder:"<ns>" help:"Call the -main function for a namespace."`
	ColorOption string `name:"color" short:"c" enum:"always,auto,none" default:"auto" placeholder:"<when>" help:"When to use colors. Possible values: always, auto, none. Defaults to auto."`
	Version     bool   `name:"version" short:"v" help:"Print the current version of Trenchman."`
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

func colorized(colorOption string) bool {
	switch colorOption {
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

func setupRepl(host string, port int, opts *repl.Opts) *repl.Repl {
	opts.In = os.Stdin
	opts.Out = os.Stdout
	opts.Err = os.Stderr
	return repl.NewRepl(opts, func(ioHandler nrepl.IOHandler) *nrepl.Client {
		client, err := nrepl.NewClient(&nrepl.Opts{
			Host:      host,
			Port:      port,
			IOHandler: ioHandler,
		})
		if err != nil {
			panic(err)
		}
		return client
	})
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
	filename := strings.TrimSpace(args.File)
	mainNS := strings.TrimSpace(args.MainNS)
	code := strings.TrimSpace(args.Eval)
	opts := &repl.Opts{
		Printer:  repl.NewPrinter(colorized(args.ColorOption)),
		HidesNil: filename != "" || mainNS != "" || code != "",
	}
	repl := setupRepl(args.Host, port, opts)
	defer repl.Close()

	if filename != "" {
		repl.Load(filename)
		return
	}
	if mainNS != "" {
		repl.Eval(fmt.Sprintf("(do (require '%s) (%s/-main))", mainNS, mainNS))
		return
	}
	if code != "" {
		repl.Eval(code)
		return
	}
	if repl.SupportsOp("interrupt") {
		repl.StartWatchingInterruption()
	}
	repl.Start()
}
