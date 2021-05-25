package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/fatih/color"
)

func startRepl(in *bufio.Reader, client *nrepl.Client) {
	for {
		fmt.Printf("%s=> ", client.CurrentNS())
		code, err := in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				client.Close()
				return
			}
			panic(err)
		}
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		result := client.Eval(code)
		if res, ok := result.(string); ok {
			color.Set(color.FgGreen)
			fmt.Println(res)
			color.Unset()
		} else if _, ok := result.(*nrepl.RuntimeError); !ok {
			panic("unexpected result received")
		}
	}
}

func startWatchInterruption(client *nrepl.Client) {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		for {
			<-interrupt
			fmt.Println("Interrupted!!")
			client.Interrupt()
		}
	}()
}

type IOHandlerImpl struct {
	r *bufio.Reader
}

func (impl *IOHandlerImpl) Out(s string) {
	color.Set(color.FgYellow)
	fmt.Print(s)
	color.Unset()
}

func (impl *IOHandlerImpl) Err(s string, fatal bool) {
	if fatal {
		panic(s)
	} else {
		color.Set(color.FgRed)
		fmt.Fprint(os.Stderr, s)
		color.Unset()
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
	startWatchInterruption(client)
	startRepl(stdin, client)
}
