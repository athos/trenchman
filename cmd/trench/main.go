package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/athos/trenchman/client"
	"github.com/athos/trenchman/nrepl"
	"github.com/athos/trenchman/prepl"
	"github.com/athos/trenchman/repl"
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
	Protocol    string `name:"protocol" short:"P" enum:"n,nrepl,p,prepl" default:"nrepl" help:"Use specified protocol. Possible values: n[repl], p[repl]. Defaults to nrepl."`
	Location    string `short:"L" help:"Connect to the specified URL (e.g. prepl://127.0.0.1:5555)"`
	Version     bool   `name:"version" short:"v" help:"Print the current version of Trenchman."`
}

var urlRegex = regexp.MustCompile(`(nrepl|prepl)://([^:]*):(\d+)`)

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

func nReplFactory(host string, port int) func(client.OutputHandler, client.ErrorHandler) client.Client {
	return func(outHandler client.OutputHandler, errHandler client.ErrorHandler) client.Client {
		c, err := nrepl.NewClient(&nrepl.Opts{
			Host:          host,
			Port:          port,
			OutputHandler: outHandler,
			ErrorHandler:  errHandler,
		})
		if err != nil {
			panic(err)
		}
		return c
	}
}

func pReplFactory(host string, port int) func(client.OutputHandler, client.ErrorHandler) client.Client {
	return func(outHandler client.OutputHandler, errHandler client.ErrorHandler) client.Client {
		c, err := prepl.NewClient(&prepl.Opts{
			Host:          host,
			Port:          port,
			OutputHandler: outHandler,
			ErrorHandler:  errHandler,
		})
		if err != nil {
			panic(err)
		}
		return c
	}
}

func setupRepl(protocol string, host string, port int, opts *repl.Opts) *repl.Repl {
	opts.In = os.Stdin
	opts.Out = os.Stdout
	opts.Err = os.Stderr
	var factory func(client.OutputHandler, client.ErrorHandler) client.Client
	switch protocol {
	case "n", "nrepl":
		factory = nReplFactory(host, port)
	case "p", "prepl":
		factory = pReplFactory(host, port)
	}
	return repl.NewRepl(opts, factory)
}

func main() {
	kong.Parse(&args)
	if args.Version {
		fmt.Printf("Trenchman %s\n", version)
		os.Exit(0)
	}

	var protocol, host string
	var port int
	loc := args.Location
	if loc != "" {
		match := urlRegex.FindStringSubmatch(loc)
		if match == nil {
			panic("bad url specified to -L option: " + loc)
		}
		protocol = match[1]
		host = match[2]
		port, _ = strconv.Atoi(match[3])
	}
	if protocol == "" && args.Protocol != "" {
		protocol = args.Protocol
	}
	if host == "" && args.Host != "" {
		host = args.Host
	}
	if port == 0 && args.Port != 0 {
		port = args.Port
	}
	if protocol == "nrepl" && port == 0 {
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
	repl := setupRepl(protocol, host, port, opts)
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
