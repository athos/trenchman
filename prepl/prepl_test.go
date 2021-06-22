package prepl

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	mockServer struct {
		steps []step
	}

	step struct {
		expected  string
		responses []string
	}
)

func (m *mockServer) Read(b []byte) (int, error) {
	if len(m.steps) == 0 {
		// when steps are all done, Read must block
		<-(chan interface{}(nil))
	}
	step := &m.steps[0]
	var buf bytes.Buffer
	for _, res := range step.responses {
		buf.WriteString(res)
	}
	n, err := buf.Read(b)
	if err != nil {
		return 0, err
	}
	m.steps = m.steps[1:]
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
	return len(b), nil
}

func (m *mockServer) Close() error {
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

func setupMock(steps ...step) *mockServer {
	s := make([]step, 1, len(steps)+1)
	s[0] = step{
		"(set! *print-namespace-maps* false)",
		[]string{`{:tag :ret, :val "nil"}`},
	}
	s = append(s, steps...)
	return &mockServer{s}
}

func TestEval(t *testing.T) {
	tests := []struct {
		input  string
		step   step
		result string
		outs   []string
		errs   []string
	}{
		{
			"(+ 1 2)",
			step{
				"(+ 1 2)\n",
				[]string{`{:tag :ret, :val "3"}`},
			},
			"3",
			nil,
			nil,
		},
		{
			"(run! prn (range 3))",
			step{
				"(run! prn (range 3))\n",
				[]string{
					`{:tag :out, :val "1"}`,
					`{:tag :out, :val "2"}`,
					`{:tag :out, :val "3"}`,
					`{:tag :ret, :val "nil"}`,
				},
			},
			"nil",
			[]string{"1", "2", "3"},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mock := setupMock(tt.step)
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
			assert.Equal(t, tt.result, ret.(string))
			assert.Nil(t, handledErr)
			assert.Equal(t, tt.outs, outHandler.outs)
			assert.Equal(t, tt.errs, outHandler.errs)
			err = c.Close()
			assert.Nil(t, err)
		})
	}
}
