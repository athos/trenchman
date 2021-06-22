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
		expected string
		response string
	}
)

func (m *mockServer) Read(b []byte) (int, error) {
	if len(m.steps) == 0 {
		// when steps are all done, Read must block
		<-(chan interface{}(nil))
	}
	step := &m.steps[0]
	bs := []byte(step.response)
	copy(b, bs)
	m.steps = m.steps[1:]
	return len(bs), nil
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

type errorHandlerFunc func(error)

func (f errorHandlerFunc) HandleErr(err error) {
	f(err)
}

func setupMock(steps []step) *mockServer {
	s := make([]step, 1, len(steps)+1)
	s[0] = step{
		"(set! *print-namespace-maps* false)",
		`{:tag :ret, :val "nil"}`,
	}
	s = append(s, steps...)
	return &mockServer{s}
}

func TestEval(t *testing.T) {
	mock := setupMock([]step{
		{
			"(+ 1 2)\n",
			`{:tag :ret, :val "3"}`,
		},
	})
	var handledErr error
	c, err := NewClient(&Opts{
		connBuilder: func(_ string, _ int) (net.Conn, error) {
			return mock, nil
		},
		ErrorHandler: errorHandlerFunc(func(err error) {
			handledErr = err
		}),
	})
	assert.Nil(t, err)
	ch := c.Eval("(+ 1 2)")
	ret := <-ch
	assert.Equal(t, "3", ret.(string))
	assert.Nil(t, handledErr)
	err = c.Close()
	assert.Nil(t, err)
}
