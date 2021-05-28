package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/athos/trenchman/trenchman/repl"
)

var opts struct {
	Host string `name:"host" help:"host" default:"127.0.0.1"`
	Port int `name:"port" required:"true" help:"port"`
}

func main() {
	kong.Parse(&opts)
	repl := repl.NewRepl(
		os.Stdin, os.Stdout, os.Stderr,
		func(ioHandler nrepl.IOHandler) *nrepl.Client {
			client, err := nrepl.NewClient(opts.Host, opts.Port, ioHandler)
			if err != nil {
				panic(err)
			}
			return client
	})
	repl.StartWatchingInterruption()
	repl.Start()
}
