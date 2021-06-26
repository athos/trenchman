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
		steps      []step
		queue      chan []string
		outs       []string
		errs       []string
		handledErr error
	}

	step struct {
		expected  string
		responses []string
	}
)

func (m *mockServer) Out(s string) {
	m.outs = append(m.outs, s)
}

func (m *mockServer) Err(s string) {
	m.errs = append(m.errs, s)
}

func (m *mockServer) HandleErr(err error) {
	m.handledErr = err
}

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

func setupClient(mock *mockServer) (*Client, error) {
	return NewClient(&Opts{
		connBuilder: func(_ string, _ int) (net.Conn, error) {
			return mock, nil
		},
		OutputHandler: mock,
		ErrorHandler:  mock,
	})
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
			c, err := setupClient(mock)
			assert.Nil(t, err)
			ch := c.Eval(tt.input)
			ret := <-ch
			assert.Equal(t, tt.result, ret)
			assert.Equal(t, tt.ns, c.CurrentNS())
			assert.Nil(t, mock.handledErr)
			assert.Equal(t, tt.outs, mock.outs)
			assert.Equal(t, tt.errs, mock.errs)
			assert.Nil(t, c.Close())
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
		c, err := setupClient(mock)
		assert.Nil(t, err)
		ch := c.Eval("(read-line)")
		go func() {
			c.Stdin("foo\n")
		}()
		ret := <-ch
		assert.Equal(t, "\"foo\"", ret)
		assert.Nil(t, mock.handledErr)
		assert.Nil(t, mock.outs)
		assert.Nil(t, mock.errs)
		assert.Nil(t, c.Close())
	})
}

func TestLoad(t *testing.T) {
	mock := setupMock([]step{
		{
			"(do (println \"Hello, World!\"))\n",
			[]string{`{:tag :ret, :val "nil"}`},
		},
	})
	c, err := setupClient(mock)
	assert.Nil(t, err)
	ch := c.Load("hello.clj", "(println \"Hello, World!\")")
	ret := <-ch
	assert.Equal(t, "nil", ret)
	assert.Nil(t, mock.handledErr)
	assert.Nil(t, mock.outs)
	assert.Nil(t, mock.errs)
	assert.Nil(t, c.Close())
}
