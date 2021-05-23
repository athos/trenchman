package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/athos/trenchman/trenchman/nrepl"
)

func startRepl(in *bufio.Reader, client *nrepl.Client) {
	for {
		fmt.Print("> ")
		code, err := in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				client.Close()
				return
			}
			panic(err)
		}
		res := client.Eval(code)
		fmt.Println(res)
	}
}

type IOHandlerImpl struct {
	r *bufio.Reader
}

func (impl *IOHandlerImpl) Out(s string) {
	fmt.Println(s)
}

func (impl *IOHandlerImpl) Err(s string, fatal bool) {
	if fatal {
		panic(s)
	} else {
		fmt.Fprintln(os.Stderr, s)
	}
}

func (impl *IOHandlerImpl) In() string {
	line, err := impl.r.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return line
}

func main() {
	stdin := bufio.NewReader(os.Stdin)
	ioHandler := &IOHandlerImpl{stdin}
	client, err := nrepl.NewClient("127.0.0.1", 49913, ioHandler)
	if err != nil {
		panic(err)
	}
	startRepl(stdin, client)
}
