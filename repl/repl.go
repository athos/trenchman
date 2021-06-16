package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"

	"github.com/athos/trenchman/client"
	"github.com/fatih/color"
)

type Repl struct {
	client   client.Client
	in       *interruptibleReader
	out      io.Writer
	err      io.Writer
	printer  Printer
	reading  atomic.Value
	hidesNil bool
}

type Opts struct {
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
	Printer  Printer
	HidesNil bool
}

func NewRepl(opts *Opts, factory func(client.OutputHandler) client.Client) *Repl {
	repl := &Repl{
		in:       newReader(opts.In),
		out:      opts.Out,
		err:      opts.Err,
		printer:  opts.Printer,
		hidesNil: opts.HidesNil,
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

func (r *Repl) SupportsOp(op string) bool {
	return r.client.SupportsOp(op)
}

func (r *Repl) Out(s string) {
	r.printer.With(color.FgYellow).Fprint(r.out, s)
}

func (r *Repl) Err(s string, fatal bool) {
	if fatal {
		panic(s)
	}
	r.printer.With(color.FgRed).Fprint(r.err, s)
}

func (r *Repl) handleResults(ch <-chan client.EvalResult) {
	for {
		select {
		case res, ok := <-ch:
			if !ok {
				return
			}
			if s, ok := res.(string); ok {
				if !r.hidesNil || s != "nil" {
					r.printer.With(color.FgGreen).Fprintln(r.out, s)
				}
			} else if _, ok := res.(*client.RuntimeError); !ok {
				panic("unexpected result received")
			}
		case res := <-r.in.readLine():
			if s, ok := res.(string); ok {
				r.client.Stdin(s)
			} else {
				switch err := res.(error); err {
				case io.EOF, errInterrupted:
				default:
					panic(err)
				}
			}
		}
	}
}

func (r *Repl) Eval(code string) {
	r.handleResults(r.client.Eval(code))
}

func (r *Repl) Load(filename string) {
	var reader *bufio.Reader
	if filename == "-" {
		reader = bufio.NewReader(os.Stdin)
	} else {
		file, err := os.Open(filename)
		if err != nil {
			panic(fmt.Errorf("cannot read file %s (%w)", filename, err))
		}
		reader = bufio.NewReader(file)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		panic(fmt.Errorf("cannot read file %s (%w)", filename, err))
	}
	r.handleResults(r.client.Load(filename, string(content)))
}

func (r *Repl) Start() {
	for {
		fmt.Fprintf(r.out, "%s=> ", r.client.CurrentNS())
		res := <-r.in.readLine()
		switch res := res.(type) {
		case error:
			switch res {
			case io.EOF, errInterrupted:
				return
			default:
				panic(res)
			}
		case string:
			code := strings.TrimSpace(res)
			if code == "" {
				continue
			}
			r.Eval(code)
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
			r.in.interrupt()
		}
	}()
}
