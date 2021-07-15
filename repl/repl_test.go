package repl

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/athos/trenchman/client"
	"github.com/stretchr/testify/assert"
)

type (
	mockClient struct {
		step step
		outs *bytes.Buffer
		errs *bytes.Buffer
	}

	step struct {
		expected string
		action   func(chan<- client.EvalResult)
	}
)

func (c *mockClient) CurrentNS() string {
	return "user"
}

func (c *mockClient) SupportsOp(op string) bool {
	return true
}

func (c *mockClient) Eval(code string) <-chan client.EvalResult {
	if code != c.step.expected {
		panic(fmt.Errorf("%s expected, but got %s", c.step.expected, code))
	}
	ch := make(chan client.EvalResult)
	go func() {
		c.step.action(ch)
		close(ch)
	}()
	return ch
}

func (c *mockClient) Load(filename string, content string) <-chan client.EvalResult {
	return nil
}

func (c *mockClient) Stdin(input string) {

}

func (c *mockClient) Interrupt() {

}

func (c *mockClient) Close() error {
	return nil
}

func newMockClient(step step) *mockClient {
	return &mockClient{
		step: step,
		outs: new(bytes.Buffer),
		errs: new(bytes.Buffer),
	}
}

type mockReader struct {
	ch chan string
}

func newMockReader(ch chan string) *mockReader {
	return &mockReader{ch}
}

func (r *mockReader) Read(bytes []byte) (int, error) {
	s, ok := <-r.ch
	if !ok {
		return 0, io.EOF
	}
	return copy(bytes, []byte(s)), nil
}

func (r *mockReader) Close() error {
	close(r.ch)
	return nil
}

func setupRepl(r *mockReader, c *mockClient) *Repl {
	return &Repl{
		client:     c,
		in:         newReader(r),
		out:        c.outs,
		err:        c.errs,
		printer:    NewMonochromePrinter(),
		lineBuffer: &lineBuffer{},
	}
}

func TestRepl(t *testing.T) {
	var repl *Repl
	var c *mockClient
	var r *mockReader
	var inputCh chan string
	tests := []struct {
		input string
		step  step
		outs  string
		errs  string
	}{
		{
			"(+ 1 2)\n",
			step{"(+ 1 2)", func(ch chan<- client.EvalResult) {
				ch <- "3"
				r.Close()
			}},
			"user=> 3\nuser=> ",
			"",
		},
		{
			"\n42\n",
			step{"42", func(ch chan<- client.EvalResult) {
				ch <- "42"
				r.Close()
			}},
			"user=> user=> 42\nuser=> ",
			"",
		},
		{
			":repl/quit\n",
			nil,
			"user=> ",
			"",
		},
		{
			"[1\n 2\n 3]\n",
			step{"[1\n 2\n 3]", func(ch chan<- client.EvalResult) {
				ch <- "[1 2 3]"
				r.Close()
			}},
			"user=>   #_=>   #_=> [1 2 3]\nuser=> ",
			"",
		},
		{
			"(println \"Hello, World!\")\n",
			step{"(println \"Hello, World!\")", func(ch chan<- client.EvalResult) {
				repl.Out("Hello, World!\n")
				ch <- "nil"
				r.Close()
			}},
			"user=> Hello, World!\nnil\nuser=> ",
			"",
		},
		{
			"(/ 1 0)\n",
			step{"(/ 1 0)", func(ch chan<- client.EvalResult) {
				repl.Err("divide by zero\n")
				ch <- client.NewRuntimeError("divide by zero")
				r.Close()
			}},
			"user=> user=> ",
			"divide by zero\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			inputCh = make(chan string, 1)
			inputCh <- tt.input
			r = newMockReader(inputCh)
			c = newMockClient(tt.step)
			repl = setupRepl(r, c)
			repl.Start()
			assert.Equal(t, tt.outs, c.outs.String())
			assert.Equal(t, tt.errs, c.errs.String())
			repl.Close()
		})
	}
}
