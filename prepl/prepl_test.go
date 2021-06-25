package prepl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/athos/trenchman/client"
	"github.com/stretchr/testify/assert"
)

type (
	mockServer struct {
		steps []step
		queue chan []string
	}

	step struct {
		expected  string
		responses []string
	}
)

func (m *mockServer) Read(b []byte) (int, error) {
	var buf bytes.Buffer
	responses, ok := <-m.queue
	if !ok {
		return 0, io.EOF
	}
	for _, res := range responses {
		buf.WriteString(res)
	}
	n, err := buf.Read(b)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (m *mockServer) Write(b []byte) (int, error) {
	if len(m.steps) == 0 {
		return 0, errors.New("expected steps to be completed")
	}
	step := &m.steps[0]
	if !bytes.Equal(b, []byte(step.expected)) {
		return 0, fmt.Errorf("%q expected, but got %q", []byte(step.expected), b)
	}
	if len(step.responses) > 0 {
		m.queue <- step.responses
	}
	m.steps = m.steps[1:]
	return len(b), nil
}

func (m *mockServer) Close() error {
	close(m.queue)
	if len(m.steps) > 0 {
		return errors.New("expected steps to be completed")
	}
	return nil
}

func (m *mockServer) LocalAddr() net.Addr                { return nil }
func (m *mockServer) RemoteAddr() net.Addr               { return nil }
func (m *mockServer) SetDeadline(t time.Time) error      { return nil }
func (m *mockServer) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockServer) SetWriteDeadline(t time.Time) error { return nil }

type mockOutputHandler struct {
	outs []string
	errs []string
}

func (m *mockOutputHandler) Out(s string) {
	m.outs = append(m.outs, s)
}

func (m *mockOutputHandler) Err(s string) {
	m.errs = append(m.errs, s)
}

type errorHandlerFunc func(error)

func (f errorHandlerFunc) HandleErr(err error) {
	f(err)
}

func setupMock(steps []step) *mockServer {
	s := make([]step, 1, len(steps)+1)
	s[0] = step{
		"(set! *print-namespace-maps* false)",
		[]string{`{:tag :ret, :val "nil"}`},
	}
	s = append(s, steps...)
	return &mockServer{
		steps: s,
		queue: make(chan []string, 1),
	}
}

func TestEval(t *testing.T) {
	tests := []struct {
		input  string
		step   step
		ns     string
		result client.EvalResult
		outs   []string
		errs   []string
	}{
		{
			"(+ 1 2)",
			step{
				"(+ 1 2)\n",
				[]string{`{:tag :ret, :val "3", :ns "user"}`},
			},
			"user",
			"3",
			nil,
			nil,
		},
		{
			"(ns foo)",
			step{
				"(ns foo)\n",
				[]string{`{:tag :ret, :val "nil", :ns "foo"}`},
			},
			"foo",
			"nil",
			nil,
			nil,
		},
		{
			"(/ 1 0)",
			step{
				"(/ 1 0)\n",
				[]string{
					`{:tag :ret, :val "{:phase :execution, :cause \"Divide by zero\", :trace [[clojure.lang.Numbers divide \"Numbers.java\" 188]], :via [{:type java.lang.ArithmeticException, :message \"Divide by zero\", :at [clojure.lang.Numbers divide \"Numbers.java\" 188]}]}", :exception true, :ns "user"}`,
				},
			},
			"user",
			client.NewRuntimeError("Execution error (ArithmeticException) at clojure.lang.Numbers/divide (Numbers.java:188).\nDivide by zero"),
			nil,
			[]string{"Execution error (ArithmeticException) at clojure.lang.Numbers/divide (Numbers.java:188).\nDivide by zero\n"},
		},
		{
			"(run! prn (range 3))",
			step{
				"(run! prn (range 3))\n",
				[]string{
					`{:tag :out, :val "1"}`,
					`{:tag :out, :val "2"}`,
					`{:tag :out, :val "3"}`,
					`{:tag :ret, :val "nil", :ns "user"}`,
				},
			},
			"user",
			"nil",
			[]string{"1", "2", "3"},
			nil,
		},
		{
			"(binding [*out* *err*] (prn 42))",
			step{
				"(binding [*out* *err*] (prn 42))\n",
				[]string{
					`{:tag :err, :val "42"}`,
					`{:tag :ret, :val "nil", :ns "user"}`,
				},
			},
			"user",
			"nil",
			nil,
			[]string{"42"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mock := setupMock([]step{tt.step})
			var outHandler mockOutputHandler
			var handledErr error
			c, err := NewClient(&Opts{
				connBuilder: func(_ string, _ int) (net.Conn, error) {
					return mock, nil
				},
				OutputHandler: &outHandler,
				ErrorHandler: errorHandlerFunc(func(err error) {
					handledErr = err
				}),
			})
			assert.Nil(t, err)
			ch := c.Eval(tt.input)
			ret := <-ch
			assert.Equal(t, tt.result, ret)
			assert.Equal(t, tt.ns, c.CurrentNS())
			assert.Nil(t, handledErr)
			assert.Equal(t, tt.outs, outHandler.outs)
			assert.Equal(t, tt.errs, outHandler.errs)
			err = c.Close()
			assert.Nil(t, err)
		})
	}
	t.Run("(read-line)", func(t *testing.T) {
		steps := []step{
			{
				"(read-line)\n",
				nil,
			},
			{
				"foo\n",
				[]string{
					`{:tag :ret, :val "\"foo\"", :ns "user"}`,
				},
			},
		}
		mock := setupMock(steps)
		var outHandler mockOutputHandler
		var handledErr error
		c, err := NewClient(&Opts{
			connBuilder: func(_ string, _ int) (net.Conn, error) {
				return mock, nil
			},
			OutputHandler: &outHandler,
			ErrorHandler: errorHandlerFunc(func(err error) {
				handledErr = err
			}),
		})
		assert.Nil(t, err)
		ch := c.Eval("(read-line)")
		go func() {
			c.Stdin("foo\n")
		}()
		ret := <-ch
		assert.Equal(t, "\"foo\"", ret)
		assert.Nil(t, handledErr)
		assert.Nil(t, outHandler.outs)
		assert.Nil(t, outHandler.errs)
		err = c.Close()
		assert.Nil(t, err)
	})
}

func TestLoad(t *testing.T) {
	mock := setupMock([]step{
		{
			"(do (println \"Hello, World!\"))\n",
			[]string{`{:tag :ret, :val "nil"}`},
		},
	})
	var outHandler mockOutputHandler
	var handledErr error
	c, err := NewClient(&Opts{
		connBuilder: func(_ string, _ int) (net.Conn, error) {
			return mock, nil
		},
		OutputHandler: &outHandler,
		ErrorHandler: errorHandlerFunc(func(err error) {
			handledErr = err
		}),
	})
	assert.Nil(t, err)
	ch := c.Load("hello.clj", "(println \"Hello, World!\")")
	ret := <-ch
	assert.Equal(t, "nil", ret)
	assert.Nil(t, handledErr)
	assert.Nil(t, outHandler.outs)
	assert.Nil(t, outHandler.errs)
	err = c.Close()
	assert.Nil(t, err)
}
