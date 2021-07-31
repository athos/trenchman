package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/athos/trenchman/client"
	"github.com/athos/trenchman/nrepl"
	"github.com/athos/trenchman/prepl"
	"github.com/athos/trenchman/repl"
	"github.com/mattn/go-isatty"
	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "v0.0.0"

const (
	COLOR_NONE   = "none"
	COLOR_AUTO   = "auto"
	COLOR_ALWAYS = "always"
)

var args = struct {
	host        *string
	port        *int
	portfile    *string
	protocol    *string
	location    *string
	eval        *string
	file        *string
	mainNS      *string
	colorOption *string
}{
	host:        kingpin.Flag("host", "Connect to the specified host. Defaults to 127.0.0.1.").PlaceHolder("HOST").Default("127.0.0.1").String(),
	port:        kingpin.Flag("port", "Connect to the specified port.").Short('p').Int(),
	portfile:    kingpin.Flag("port-file", "Specify port file that specifies port to connect to. Defaults to .nrepl-port.").String(),
	protocol:    kingpin.Flag("protocol", "Use the specified protocol. Possible values: n[repl], p[repl]. Defaults to nrepl.").Default("nrepl").Short('P').Enum("n", "nrepl", "p", "prepl"),
	location:    kingpin.Flag("connect", "Connect to the specified URL (e.g. prepl://127.0.0.1:5555).").Short('c').PlaceHolder("URL").String(),
	eval:        kingpin.Flag("eval", "Evaluate an expression.").Short('e').PlaceHolder("EXPR").String(),
	file:        kingpin.Flag("file", "Evaluate a file.").Short('f').String(),
	mainNS:      kingpin.Flag("main", "Call the -main function for a namespace.").Short('m').PlaceHolder("NAMESPACE").String(),
	colorOption: kingpin.Flag("color", "When to use colors. Possible values: always, auto, none. Defaults to auto.").Default(COLOR_AUTO).Short('C').Enum(COLOR_NONE, COLOR_AUTO, COLOR_ALWAYS),
}

var urlRegex = regexp.MustCompile(`(nrepl|prepl)://([^:]*):(\d+)`)

var portfileNotSpecified = errors.New("port file not specified")

func readPortFromFile(protocol, portFile string) (int, error) {
	filename := portFile
	if portFile == "" {
		if protocol == "nrepl" {
			filename = ".nrepl-port"
		} else {
			filename = ".prepl-port"
		}
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		if portFile == "" {
			return 0, portfileNotSpecified
		}
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
	if protocol == "nrepl" {
		factory = nReplFactory(host, port)
	} else {
		factory = pReplFactory(host, port)
	}
	return repl.NewRepl(opts, factory)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	var protocol, host string
	var port int
	loc := *args.location
	if loc != "" {
		match := urlRegex.FindStringSubmatch(loc)
		if match == nil {
			panic("bad url specified to -L option: " + loc)
		}
		protocol = match[1]
		host = match[2]
		port, _ = strconv.Atoi(match[3])
	}
	if protocol == "" {
		switch *args.protocol {
		case "n", "nrepl":
			protocol = "nrepl"
		case "p", "prepl":
			protocol = "prepl"
		}
	}
	if host == "" && *args.host != "" {
		host = *args.host
	}
	if port == 0 && *args.port != 0 {
		port = *args.port
	}
	if port == 0 {
		p, err := readPortFromFile(protocol, *args.portfile)
		if err != nil {
			errmsg := "Port must be specified with -p or -L"
			if err != portfileNotSpecified {
				errmsg = fmt.Sprintf("Could not read port file (%s)", *args.portfile)
			}
			fmt.Fprintln(os.Stderr, errmsg)
			os.Exit(1)
		}
		port = p
	}
	filename := strings.TrimSpace(*args.file)
	mainNS := strings.TrimSpace(*args.mainNS)
	code := strings.TrimSpace(*args.eval)
	opts := &repl.Opts{
		Printer:  repl.NewPrinter(colorized(*args.colorOption)),
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
