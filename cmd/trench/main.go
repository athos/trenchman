package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/athos/trenchman/client"
	"github.com/athos/trenchman/repl"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"gopkg.in/alecthomas/kingpin.v2"
)

var version = "0.0.0"

const (
	COLOR_NONE   = "none"
	COLOR_AUTO   = "auto"
	COLOR_ALWAYS = "always"
)

type cmdArgs struct {
	port        *int
	portfile    *string
	protocol    *string
	server      *string
	init        *string
	eval        *string
	file        *string
	mainNS      *string
	initNS      *string
	colorOption *string
	args        *[]string
}

type errorHandler struct {
	printer repl.Printer
}

func (h errorHandler) HandleErr(err error) {
	var errmsg string
	switch err {
	case client.ErrDisconnected:
		errmsg = "disconnected from server"
	default:
		errmsg = err.Error()
	}
	h.printer.With(color.FgRed).Fprintln(os.Stderr, errmsg)
	os.Exit(1)
}

var args = cmdArgs{
	port:        kingpin.Flag("port", "Connect to the specified port.").Short('p').Int(),
	portfile:    kingpin.Flag("port-file", "Specify port file that specifies port to connect to. Defaults to .nrepl-port.").PlaceHolder("FILE").String(),
	protocol:    kingpin.Flag("protocol", "Use the specified protocol. Possible values: n[repl], p[repl]. Defaults to nrepl.").Default("nrepl").Short('P').Enum("n", "nrepl", "p", "prepl"),
	server:      kingpin.Flag("server", "Connect to the specified URL (e.g. prepl://127.0.0.1:5555).").Default("127.0.0.1").Short('s').PlaceHolder("[(nrepl|prepl)://]host[:port]").String(),
	init:        kingpin.Flag("init", "Load a file before execution.").Short('i').PlaceHolder("FILE").String(),
	eval:        kingpin.Flag("eval", "Evaluate an expression.").Short('e').PlaceHolder("EXPR").String(),
	file:        kingpin.Flag("file", "Evaluate a file.").Short('f').String(),
	mainNS:      kingpin.Flag("main", "Call the -main function for a namespace.").Short('m').PlaceHolder("NAMESPACE").String(),
	initNS:      kingpin.Flag("init-ns", "Initialize REPL with the specified namespace. Defaults to \"user\".").PlaceHolder("NAMESPACE").String(),
	colorOption: kingpin.Flag("color", "When to use colors. Possible values: always, auto, none. Defaults to auto.").Default(COLOR_AUTO).Short('C').Enum(COLOR_NONE, COLOR_AUTO, COLOR_ALWAYS),
	args:        kingpin.Arg("args", "Arguments to pass to -main. These will be ignored unless -m is specified.").Strings(),
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

func buildMainInvocation(mainNS string, args []string) string {
	quotedArgs := []string{}
	for _, arg := range args {
		quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
	}
	argStr := strings.Join(quotedArgs, " ")
	return fmt.Sprintf("(do (require '%s) (%s/-main %s) nil)", mainNS, mainNS, argStr)
}

func main() {
	kingpin.Version("Trenchman " + version)
	kingpin.Parse()

	printer := repl.NewPrinter(colorized(*args.colorOption))
	errHandler := errorHandler{printer}
	helper := setupHelper{errHandler}
	protocol, connBuilder := helper.resolveConnection(&args)
	initFile := strings.TrimSpace(*args.init)
	filename := strings.TrimSpace(*args.file)
	initNS := strings.TrimSpace(*args.initNS)
	mainNS := strings.TrimSpace(*args.mainNS)
	code := strings.TrimSpace(*args.eval)
	opts := &repl.Opts{
		Printer:  printer,
		HidesNil: filename != "" || mainNS != "" || code != "",
	}
	repl := helper.setupRepl(protocol, connBuilder, initNS, opts)
	defer repl.Close()

	if initFile != "" {
		repl.LoadWithResultVisibility(initFile, true)
	}
	if filename != "" {
		repl.Load(filename)
		return
	}
	if mainNS != "" {
		repl.Eval(buildMainInvocation(mainNS, *args.args))
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
