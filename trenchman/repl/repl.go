package repl

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"

	"github.com/athos/trenchman/trenchman/nrepl"
	"github.com/fatih/color"
)

type Repl struct {
	client   *nrepl.Client
	in       *interruptibleReader
	out      io.Writer
	err      io.Writer
	cancel   chan<- struct{}
	reading  atomic.Value
	hidesNil bool
}

func NewRepl(
	in io.ReadCloser,
	out io.Writer,
	err io.Writer,
	hidesNil bool,
	factory func(nrepl.IOHandler) *nrepl.Client,
) *Repl {
	ch := make(chan struct{}, 1)
	repl := &Repl{
		in:     newReader(ch, in),
		out:    out,
		err:    err,
		cancel: ch,
		hidesNil: hidesNil,
	}
	client := factory(repl)
	repl.client = client
	repl.reading.Store(false)
	return repl
}

func (r *Repl) Close() error {
	if err := r.in.Close(); err != nil {
		return err
	}
	return r.client.Close()
}

func (r *Repl) readLine() (string, error) {
	r.reading.Store(true)
	ret, err := r.in.readLine()
	r.reading.Store(false)
	return ret, err
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
	line, err := r.readLine()
	if err != nil {
		if err == errInterrupted {
			return "", false
		}
		panic(err)
	}
	return line, true
}

func (r *Repl) Eval(code string) {
	for res := range r.client.Eval(code) {
		if s, ok := res.(string); ok {
			if !r.hidesNil || s != "nil" {
				color.Set(color.FgGreen)
				fmt.Fprintln(r.out, s)
				color.Unset()
			}
		} else if _, ok := res.(*nrepl.RuntimeError); !ok {
			panic("unexpected result received")
		}
	}
}

func (r *Repl) Start() {
	defer r.Close()
	for {
		fmt.Fprintf(r.out, "%s=> ", r.client.CurrentNS())
		code, err := r.readLine()
		if err != nil {
			switch err {
			case io.EOF:
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
		r.Eval(code)
	}
}

func (r *Repl) StartWatchingInterruption() {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		for {
			<-interrupt
			r.client.Interrupt()
			if r.reading.Load().(bool) {
				r.cancel <- struct{}{}
			}
		}
	}()
}
