package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

type (
	MockServer struct {
		steps      []Step
		queue      chan []string
		outs       []string
		errs       []string
		handledErr error
	}

	Step struct {
		Expected  string
		Responses []string
	}
)

func (m *MockServer) Outs() []string {
	return m.outs
}

func (m *MockServer) Errs() []string {
	return m.errs
}

func (m *MockServer) HandledErr() error {
	return m.handledErr
}

func (m *MockServer) Out(s string) {
	m.outs = append(m.outs, s)
}

func (m *MockServer) Err(s string) {
	m.errs = append(m.errs, s)
}

func (m *MockServer) Debug(s string) {}

func (m *MockServer) HandleErr(err error) {
	m.handledErr = err
}

func (m *MockServer) Read(b []byte) (int, error) {
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

func (m *MockServer) Write(b []byte) (int, error) {
	if len(m.steps) == 0 {
		return 0, errors.New("expected steps to be completed")
	}
	step := &m.steps[0]
	if !bytes.Equal(b, []byte(step.Expected)) {
		return 0, fmt.Errorf("%q expected, but got %q", []byte(step.Expected), b)
	}
	if len(step.Responses) > 0 {
		m.queue <- step.Responses
	}
	m.steps = m.steps[1:]
	return len(b), nil
}

func (m *MockServer) Close() error {
	close(m.queue)
	if len(m.steps) > 0 {
		return errors.New("expected steps to be completed")
	}
	return nil
}

func (m *MockServer) LocalAddr() net.Addr                { return nil }
func (m *MockServer) RemoteAddr() net.Addr               { return nil }
func (m *MockServer) SetDeadline(t time.Time) error      { return nil }
func (m *MockServer) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockServer) SetWriteDeadline(t time.Time) error { return nil }

func NewMockServer(steps []Step) *MockServer {
	return &MockServer{
		steps: steps,
		queue: make(chan []string, 1),
	}
}
