package nrepl

import (
	"fmt"
	"sync/atomic"

	"github.com/athos/trenchman/trenchman/bencode"
	"github.com/google/uuid"
)

type (
	IOHandler interface {
		In() string
		Out(s string)
		Err(s string, fatal bool)
	}

	Client struct {
		conn      *Conn
		session   Session
		ns        atomic.Value
		ch        chan EvalResult
		ioHandler IOHandler
		done      chan struct{}
		pending   *pending
	}

	Session string
	pending struct {
		id string
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
	client.session = Session(session)
	client.ns.Store("user")
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

func (c *Client) CurrentNS() string {
	return c.ns.Load().(string)
}

func has(resp Response, key string) bool {
	_, ok := resp[key]
	return ok
}

func (c *Client) statusContains(datum bencode.Datum, status string) bool {
	statuses, ok := datum.([]bencode.Datum)
	if !ok {
		msg := fmt.Sprintf("Unknown status returned: %v", statuses)
		c.ioHandler.Err(msg, true)
	}
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

func (c *Client) handleResp(resp Response) {
	//fmt.Printf("RESP: %v\n", resp)
	switch {
	case has(resp, "value"):
		c.ns.Store(resp["ns"].(string))
		c.ch <- resp["value"].(string)
	case has(resp, "ex"):
		c.ch <- &RuntimeError{resp["ex"].(string)}
	case has(resp, "out"):
		c.ioHandler.Out(resp["out"].(string))
	case has(resp, "err"):
		c.ioHandler.Err(resp["err"].(string), false)
	case has(resp, "status"):
		status := resp["status"]
		if c.statusContains(status, "need-input") {
			c.stdin(c.ioHandler.In())
		} else if c.statusContains(status, "done") {
			if has(resp, "id") && c.pending.id == resp["id"].(string) {
				c.pending = nil
			}
		}
	default:
		msg := fmt.Sprintf("Unknown response returned: %v", resp)
		c.ioHandler.Err(msg, true)
	}
}

func (c *Client) send(req Request) {
	req["session"] = string(c.session)
	if err := c.conn.sendReq(req); err != nil {
		c.ioHandler.Err(err.Error(), true)
	}
}

func (c *Client) Eval(code string) EvalResult {
	id := uuid.NewString()
	c.pending = &pending{id}
	c.send(Request{
		"op":   "eval",
		"id":   id,
		"code": code,
		"ns":   c.CurrentNS(),
	})
	return <-c.ch
}

func (c *Client) stdin(in string) {
	c.send(Request{
		"op":    "stdin",
		"stdin": in,
	})
}

func (c *Client) Interrupt() {
	if c.pending != nil {
		c.send(Request{
			"op": "interrupt",
			"interrupt-id": c.pending.id,
		})
	}
}
