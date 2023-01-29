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

var urlRegex = regexp.MustCompile(`^(?:(nrepl|prepl)://)?([^:]+)(?::(\d+))?$`)
var unixUrlRegex = regexp.MustCompile(`^nrepl\+unix:(.+)$`)

type setupHelper struct {
	errHandler client.ErrorHandler
	debug      bool
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

func (h setupHelper) nReplFactory(connBuilder client.ConnBuilder, initNS string, oneshot bool) func(client.OutputHandler) client.Client {
	return func(outHandler client.OutputHandler) client.Client {
		c, err := nrepl.NewClient(&nrepl.Opts{
			ConnBuilder:   connBuilder,
			InitNS:        initNS,
			OutputHandler: outHandler,
			ErrorHandler:  h.errHandler,
			Oneshot:       oneshot,
			Debug:         h.debug,
		})
		if err != nil {
			h.errHandler.HandleErr(err)
		}
		return c
	}
}

func (h setupHelper) pReplFactory(connBuilder client.ConnBuilder, initNS string, oneshot bool) func(client.OutputHandler) client.Client {
	return func(outHandler client.OutputHandler) client.Client {
		c, err := prepl.NewClient(&prepl.Opts{
			ConnBuilder:   connBuilder,
			InitNS:        initNS,
			OutputHandler: outHandler,
			ErrorHandler:  h.errHandler,
			Oneshot:       oneshot,
			Debug:         h.debug,
		})
		if err != nil {
			h.errHandler.HandleErr(err)
		}
		return c
	}
}

func (h setupHelper) setupRepl(protocol string, connBuilder client.ConnBuilder, initNS string, oneshot bool, opts *repl.Opts) *repl.Repl {
	opts.In = os.Stdin
	opts.Out = os.Stdout
	opts.Err = os.Stderr
	opts.ErrHandler = h.errHandler
	var factory func(client.OutputHandler) client.Client
	if protocol == "nrepl" {
		factory = h.nReplFactory(connBuilder, initNS, oneshot)
	} else {
		factory = h.pReplFactory(connBuilder, initNS, oneshot)
	}
	return repl.NewRepl(opts, factory)
}

func (h setupHelper) parseServerUrl(url string) (protocol string, dest string, port int, err error) {
	if match := urlRegex.FindStringSubmatch(url); match != nil {
		protocol = match[1]
		dest = match[2]
		if match[3] != "" {
			port, _ = strconv.Atoi(match[3])
		}
		return
	}
	if match := unixUrlRegex.FindStringSubmatch(url); match != nil {
		protocol = "nrepl+unix"
		dest = match[1]
		return
	}
	err = errors.New("bad url specified to -s option: " + url)
	return
}

func (h setupHelper) resolveProtocol(protocol string, args *cmdArgs) (ret string, unixSocket bool) {
	if protocol == "" {
		protocol = *args.protocol
	}
	switch protocol {
	case "n", "nrepl":
		ret = "nrepl"
	case "nrepl+unix":
		ret = "nrepl"
		unixSocket = true
	case "p", "prepl":
		ret = "prepl"
	}
	return
}

func (h setupHelper) resolvePort(protocol string, port int, args *cmdArgs) (int, error) {
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
			return 0, err
		}
		port = p
	}
	return port, nil
}

func (h setupHelper) resolveConnection(args *cmdArgs) (protocol string, connBuilder client.ConnBuilder) {
	server := *args.server
	var dest string
	var port int
	if server != "" {
		var err error
		if protocol, dest, port, err = h.parseServerUrl(server); err != nil {
			h.errHandler.HandleErr(err)
			return
		}
	}
	protocol, unixSocket := h.resolveProtocol(protocol, args)
	if unixSocket {
		connBuilder = &client.UnixConnBuilder{Path: dest}
	} else {
		port, err := h.resolvePort(protocol, port, args)
		if err != nil {
			h.errHandler.HandleErr(err)
			return
		}
		connBuilder = &client.TCPConnBuilder{Host: dest, Port: port}
	}
	if *args.retryTimeout > 0 {
		connBuilder = client.NewRetryConnBuilder(connBuilder, *args.retryTimeout, *args.retryInterval)
	}
	return
}
