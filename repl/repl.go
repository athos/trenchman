package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/athos/trenchman/client"
	"github.com/fatih/color"
)

type Repl struct {
	client     client.Client
	in         *interruptibleReader
	out        io.Writer
	err        io.Writer
	printer    Printer
	errHandler client.ErrorHandler
	lineBuffer *lineBuffer
	hidesNil   bool
}

type Opts struct {
	In         io.Reader
	Out        io.Writer
	Err        io.Writer
	Printer    Printer
	ErrHandler client.ErrorHandler
	HidesNil   bool
}

func NewRepl(
	opts *Opts,
	factory func(client.OutputHandler) client.Client,
) *Repl {
	repl := &Repl{
		in:         newReader(opts.In),
		out:        opts.Out,
		err:        opts.Err,
		printer:    opts.Printer,
		errHandler: opts.ErrHandler,
		lineBuffer: &lineBuffer{},
		hidesNil:   opts.HidesNil,
	}
	client := factory(repl)
	repl.client = client
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

func (r *Repl) Err(s string) {
	r.printer.With(color.FgRed).Fprint(r.err, s)
}

func (r *Repl) handleResults(ch <-chan client.EvalResult, hidesResult bool) {
	for {
		select {
		case res, ok := <-ch:
			if !ok {
				return
			}
			if s, ok := res.(string); ok {
				if !hidesResult && (!r.hidesNil || s != "nil") {
					fmt.Fprintln(r.out, s)
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
					r.errHandler.HandleErr(err)
				}
			}
		}
	}
}

func (r *Repl) Eval(code string) {
	r.handleResults(r.client.Eval(code), false)
}

func (r *Repl) LoadWithResultVisibility(filename string, hidesResult bool) {
	var reader *bufio.Reader
	if filename == "-" {
		reader = bufio.NewReader(os.Stdin)
	} else {
		file, err := os.Open(filename)
		if err != nil {
			r.errHandler.HandleErr(err)
		}
		reader = bufio.NewReader(file)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		r.errHandler.HandleErr(err)
	}
	r.handleResults(r.client.Load(filename, string(content)), hidesResult)
}

func (r *Repl) Load(filename string) {
	r.LoadWithResultVisibility(filename, false)
}

func (r *Repl) Interrupt() {
	r.client.Interrupt()
	r.in.interrupt()
}

func (r *Repl) Start() {
	continued := false
	for {
		if continued {
			prompt := strings.Repeat(" ", len(r.client.CurrentNS())-2) + "#_=> "
			r.printer.With(color.FgGreen).Fprint(r.out, prompt)
		} else {
			r.printer.With(color.FgGreen).Fprintf(r.out, "%s=> ", r.client.CurrentNS())
		}
		res := <-r.in.readLine()
		switch res := res.(type) {
		case error:
			switch res {
			case errInterrupted:
				if continued {
					r.lineBuffer.reset()
					continued = false
					fmt.Fprintln(r.out)
					continue
				}
				return
			case io.EOF:
				return
			default:
				r.errHandler.HandleErr(res)
			}
		case string:
			s, cont, _ := r.lineBuffer.feedLine(res)
			continued = cont
			if cont {
				continue
			}
			code := strings.TrimSpace(s)
			switch code {
			case "":
				continue
			case ":repl/quit":
				return
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
			r.Interrupt()
		}
	}()
}
