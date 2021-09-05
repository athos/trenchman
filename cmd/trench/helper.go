package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/athos/trenchman/client"
	"github.com/athos/trenchman/nrepl"
	"github.com/athos/trenchman/prepl"
	"github.com/athos/trenchman/repl"
)

var urlRegex = regexp.MustCompile(`(?:(nrepl|prepl)://)?([^:]+)(?::(\d+))?`)

type setupHelper struct {
	errHandler client.ErrorHandler
}

func readPortFromFile(protocol, portFile string) (int, bool, error) {
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
			return 0, false, err
		}
		return 0, true, err
	}
	port, err := strconv.Atoi(string(content))
	if err != nil {
		return 0, false, err
	}
	return port, false, nil
}

func (h setupHelper) nReplFactory(connBuilder client.ConnBuilder, initNS string) func(client.OutputHandler) client.Client {
	return func(outHandler client.OutputHandler) client.Client {
		c, err := nrepl.NewClient(&nrepl.Opts{
			ConnBuilder:   connBuilder,
			InitNS:        initNS,
			OutputHandler: outHandler,
			ErrorHandler:  h.errHandler,
		})
		if err != nil {
			h.errHandler.HandleErr(err)
		}
		return c
	}
}

func (h setupHelper) pReplFactory(connBuilder client.ConnBuilder, initNS string) func(client.OutputHandler) client.Client {
	return func(outHandler client.OutputHandler) client.Client {
		c, err := prepl.NewClient(&prepl.Opts{
			ConnBuilder:   connBuilder,
			InitNS:        initNS,
			OutputHandler: outHandler,
			ErrorHandler:  h.errHandler,
		})
		if err != nil {
			h.errHandler.HandleErr(err)
		}
		return c
	}
}

func (h setupHelper) setupRepl(protocol string, connBuilder client.ConnBuilder, initNS string, opts *repl.Opts) *repl.Repl {
	opts.In = os.Stdin
	opts.Out = os.Stdout
	opts.Err = os.Stderr
	opts.ErrHandler = h.errHandler
	var factory func(client.OutputHandler) client.Client
	if protocol == "nrepl" {
		factory = h.nReplFactory(connBuilder, initNS)
	} else {
		factory = h.pReplFactory(connBuilder, initNS)
	}
	return repl.NewRepl(opts, factory)
}

func (h setupHelper) resolveConnection(args *cmdArgs) (protocol string, connBuilder client.ConnBuilder) {
	server := *args.server
	var host string
	var port int
	if server != "" {
		match := urlRegex.FindStringSubmatch(server)
		if match == nil {
			err := errors.New("bad url specified to -s option: " + server)
			h.errHandler.HandleErr(err)
			return
		}
		protocol = match[1]
		host = match[2]
		if match[3] != "" {
			port, _ = strconv.Atoi(match[3])
		}
	}
	if protocol == "" {
		switch *args.protocol {
		case "n", "nrepl":
			protocol = "nrepl"
		case "p", "prepl":
			protocol = "prepl"
		}
	}
	if port == 0 && *args.port != 0 {
		port = *args.port
	}
	if port == 0 {
		p, portfileSpecified, err := readPortFromFile(protocol, *args.portfile)
		if err != nil {
			if portfileSpecified {
				err = fmt.Errorf("could not read port file: %s", *args.portfile)
			} else {
				err = errors.New("port must be specified with -p or -s")
			}
			h.errHandler.HandleErr(err)
			return
		}
		port = p
	}
	connBuilder = &client.TCPConnBuilder{Host: host, Port: port}
	return
}
