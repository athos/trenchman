package client

import (
	"errors"
	"io"
)

type (
	Request  interface{}
	Response interface{}

	Conn interface {
		io.Closer
		Send(req Request) error
		Recv() (Response, error)
	}

	Handler interface {
		HandleResp(resp Response)
		HandleErr(error)
	}

	// EvalResult is either string or RuntimeError
	EvalResult   interface{}
	RuntimeError struct {
		err string
	}

	Client interface {
		io.Closer
		CurrentNS() string
		SupportsOp(op string) bool
		Eval(code string) <-chan EvalResult
		Load(filename string, content string) <-chan EvalResult
		Stdin(input string)
		Interrupt()
	}

	OutputHandler interface {
		Out(s string)
		Err(s string, fatal bool)
	}
)

var ErrDisconnected = errors.New("disconnected")

func NewRuntimeError(err string) *RuntimeError {
	return &RuntimeError{err}
}

func (e *RuntimeError) Error() string {
	return e.err
}

func StartLoop(conn Conn, handler Handler, done chan struct{}) {
	for {
		resp, err := conn.Recv()
		if err != nil {
			select {
			case <-done:
				return
			default:
				if err != nil {
					handler.HandleErr(err)
					return
				}
			}
		}
		handler.HandleResp(resp)
	}
}
