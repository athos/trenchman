package nrepl

import (
	"fmt"
)

type (
	IOHandler interface {
		In() string
		Out(s string)
		Err(s string, fatal bool)
	}

	Client struct {
		conn      *Conn
		session   string
		ch        chan EvalResult
		ioHandler IOHandler
		done      chan struct{}
	}

	// EvalResult is either string or RuntimeError
	EvalResult interface{}

	RuntimeError struct {
		err string
	}
)

func (e *RuntimeError) Error() string {
	return e.err
}

func NewClient(host string, port int, ioHandler IOHandler) (*Client, error) {
	client := &Client{
		ch:        make(chan EvalResult),
		ioHandler: ioHandler,
		done:      make(chan struct{}),
	}
	builder := NewBuilder(host, port)
	builder.Handler = func(r Response) { client.handleResp(r) }
	builder.ErrHandler = func(err error) {
		ioHandler.Err(err.Error(), true)
	}
	conn, err := builder.Connect()
	if err != nil {
		return nil, err
	}
	session, err := conn.initSession()
	if err != nil {
		return nil, err
	}
	client.conn = conn
	client.session = session
	go conn.startLoop(client.done)
	return client, nil
}

func (c *Client) Close() error {
	close(c.done)
	if err := c.conn.Close(); err != nil {
		return err
	}
	close(c.ch)
	return nil
}

func has(resp Response, key string) bool {
	_, ok := resp[key]
	return ok
}

func (c *Client) handleResp(resp Response) {
	switch {
	case has(resp, "value"):
		c.ch <- resp["value"].(string)
	case has(resp, "ex"):
		c.ch <- &RuntimeError{resp["ex"].(string)}
	case has(resp, "out"):
		c.ioHandler.Out(resp["out"].(string))
	case has(resp, "err"):
		c.ioHandler.Err(resp["err"].(string), false)
	case !has(resp, "status"):
		msg := fmt.Sprintf("Unknown response returned: %v", resp)
		c.ioHandler.Err(msg, true)
	}
}

func (c *Client) Eval(code string) EvalResult {
	req := Request{
		"op":      "eval",
		"code":    code,
		"session": c.session,
	}
	err := c.conn.sendReq(req)
	if err != nil {
		c.ioHandler.Err(err.Error(), true)
	}
	return <-c.ch
}
