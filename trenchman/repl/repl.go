package repl

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

type Repl struct {
	client *nrepl.Client
	in     *bufio.Reader
	out    io.Writer
	err    io.Writer
}

func NewRepl(
	in io.Reader,
	out io.Writer,
	err io.Writer,
	factory func(nrepl.IOHandler) *nrepl.Client,
) *Repl {
	repl := &Repl{
		in:  bufio.NewReader(in),
		out: out,
		err: err,
	}
	client := factory(repl)
	repl.client = client
	return repl
}

func (r *Repl) Out(s string) {
	color.Set(color.FgYellow)
	fmt.Fprint(r.out, s)
	color.Unset()
}

func (r *Repl) Err(s string, fatal bool) {
	if fatal {
		panic(s)
	} else {
		color.Set(color.FgRed)
		fmt.Fprint(r.err, s)
		color.Unset()
	}
}

func (r *Repl) In() string {
	line, err := r.in.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return line
}

func (r *Repl) Start() {
	for {
		fmt.Fprintf(r.out, "%s=> ", r.client.CurrentNS())
		code, err := r.in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				r.client.Close()
				return
			}
			panic(err)
		}
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		result := r.client.Eval(code)
		if res, ok := result.(string); ok {
			color.Set(color.FgGreen)
			fmt.Fprintln(r.out, res)
			color.Unset()
		} else if _, ok := result.(*nrepl.RuntimeError); !ok {
			panic("unexpected result received")
		}
	}
}

func (r *Repl) StartWatchingInterruption() {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		for {
			<-interrupt
			r.client.Interrupt()
		}
	}()
}
