package main

import (
	"os"

	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/athos/trenchman/trenchman/repl"
)

func main() {
	repl := repl.NewRepl(
		os.Stdin, os.Stdout, os.Stderr,
		func(ioHandler nrepl.IOHandler) *nrepl.Client {
			client, err := nrepl.NewClient("127.0.0.1", 59800, ioHandler)
			if err != nil {
				panic(err)
			}
			return client
	})
	repl.StartWatchingInterruption()
	repl.Start()
}
