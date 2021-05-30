package nrepl

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/athos/trenchman/trenchman/bencode"
	"github.com/google/uuid"
)

type (
	IOHandler interface {
		In() (ret string, ok bool)
		Out(s string)
		Err(s string, fatal bool)
	}

	Client struct {
		conn        *Conn
		lock        sync.RWMutex
		sessionInfo *SessionInfo
		ns          string
		ioHandler   IOHandler
		done        chan struct{}
		pending     map[string]chan EvalResult
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
		ns:        "user",
		ioHandler: ioHandler,
		done:      make(chan struct{}),
		pending:   map[string]chan EvalResult{},
	}
	opts := &ConnOpts{
		Host:    host,
		Port:    port,
		Handler: func(r Response) { client.handleResp(r) },
		ErrHandler: func(err error) {
			ioHandler.Err(err.Error(), true)
		},
	}
	conn, err := Connect(opts)
	if err != nil {
		return nil, err
	}
	sessionInfo, err := conn.initSession()
	if err != nil {
		return nil, err
	}
	client.conn = conn
	client.sessionInfo = sessionInfo
	go conn.startLoop(client.done)
	return client, nil
}

func (c *Client) Close() error {
	close(c.done)
	if err := c.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (c *Client) CurrentNS() string {
	return c.ns
}

func (c *Client) SupportsOp(op string) bool {
	if c.sessionInfo == nil {
		return false
	}
	_, ok := c.sessionInfo.ops[op]
	return ok
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
		id := resp["id"].(string)
		c.lock.Lock()
		ch := c.pending[id]
		if has(resp, "ns") {
			c.ns = resp["ns"].(string)
		}
		c.lock.Unlock()
		ch <- resp["value"].(string)
	case has(resp, "ex"):
		id := resp["id"].(string)
		c.lock.RLock()
		ch := c.pending[id]
		c.lock.RUnlock()
		ch <- &RuntimeError{resp["ex"].(string)}
	case has(resp, "out"):
		c.ioHandler.Out(resp["out"].(string))
	case has(resp, "err"):
		c.ioHandler.Err(resp["err"].(string), false)
	case has(resp, "status"):
		c.handleStatusUpdate(resp)
	default:
		msg := fmt.Sprintf("Unknown response returned: %v", resp)
		c.ioHandler.Err(msg, true)
	}
}

func (c *Client) handleStatusUpdate(resp Response) {
	status := resp["status"]
	if c.statusContains(status, "need-input") {
		input, ok := c.ioHandler.In()
		// If not ok, input request must have been cancelled by user
		// So, then nothing to do more
		if ok {
			c.stdin(input)
		}
	} else if c.statusContains(status, "done") {
		if has(resp, "id") {
			id := resp["id"].(string)
			c.lock.Lock()
			ch := c.pending[id]
			delete(c.pending, id)
			c.lock.Unlock()
			close(ch)
		}
	}
}

func (c *Client) send(req Request) {
	if c.sessionInfo != nil {
		req["session"] = c.sessionInfo.session
	}
	if err := c.conn.sendReq(req); err != nil {
		c.ioHandler.Err(err.Error(), true)
	}
}

func (c *Client) newIdChan() (string, chan EvalResult) {
	id := uuid.NewString()
	ch := make(chan EvalResult)
	c.lock.Lock()
	c.pending[id] = ch
	c.lock.Unlock()
	return id, ch
}

func (c *Client) Eval(code string) <-chan EvalResult {
	id, ch := c.newIdChan()
	c.send(Request{
		"op":   "eval",
		"id":   id,
		"code": code,
		"ns":   c.CurrentNS(),
	})
	return ch
}

func (c *Client) Load(filename string, content string) <-chan EvalResult {
	id, ch := c.newIdChan()
	base := filepath.Base(filename)
	dir := filepath.Dir(filename)
	c.send(Request{
		"op":        "load-file",
		"id":        id,
		"ns":        c.CurrentNS(),
		"file":      content,
		"file-name": base,
		"file-path": dir,
	})
	return ch
}

func (c *Client) stdin(in string) {
	c.send(Request{
		"op":    "stdin",
		"stdin": in,
	})
}

func (c *Client) Interrupt() {
	ids := []string{}
	c.lock.RLock()
	for id := range c.pending {
		ids = append(ids, id)
	}
	c.lock.RUnlock()
	for _, id := range ids {
		c.send(Request{
			"op":           "interrupt",
			"interrupt-id": id,
		})
	}
}
