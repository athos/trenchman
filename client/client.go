package client

import (
	"errors"
	"io"
)

type (
	Request  interface{}
	Response interface{}

	Transport interface {
		io.Closer
		Send(req Request) error
		Recv() (Response, error)
	}

	ResponseHandler interface {
		HandleResp(resp Response)
	}

	ErrorHandler interface {
		HandleErr(error)
	}

	Handler interface {
		ResponseHandler
		ErrorHandler
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
		Err(s string)
		Debug(s string)
	}
)

var ErrDisconnected = errors.New("disconnected")

func NewRuntimeError(err string) *RuntimeError {
	return &RuntimeError{err}
}

func (e *RuntimeError) Error() string {
	return e.err
}

func StartLoop(transport Transport, handler Handler, done chan struct{}) {
	for {
		resp, err := transport.Recv()
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
