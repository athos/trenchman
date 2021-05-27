package repl

import (
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
	in     *interruptibleReader
	out    io.Writer
	err    io.Writer
	cancel chan struct{}
}

func NewRepl(
	in io.ReadCloser,
	out io.Writer,
	err io.Writer,
	factory func(nrepl.IOHandler) *nrepl.Client,
) *Repl {
	ch := make(chan struct{}, 1)
	repl := &Repl{
		in:     newReader(ch, in),
		out:    out,
		err:    err,
		cancel: ch,
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

func (r *Repl) In() (string, bool) {
	line, err := r.in.ReadLine()
	if err != nil {
		if err == errInterrupted {
			return "", false
		}
		panic(err)
	}
	return line, true
}

func (r *Repl) Start() {
	for {
		fmt.Fprintf(r.out, "%s=> ", r.client.CurrentNS())
		code, err := r.in.ReadLine()
		if err != nil {
			switch err {
			case io.EOF:
				r.client.Close()
				return
			case errInterrupted:
				continue
			default:
				panic(err)
			}
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
			r.cancel <- struct{}{}
		}
	}()
}
