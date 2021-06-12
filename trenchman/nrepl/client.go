package nrepl

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/athos/trenchman/trenchman/bencode"
	"github.com/athos/trenchman/trenchman/client"
	"github.com/google/uuid"
)

type (
	Client struct {
		conn           *Conn
		sessionInfo    *SessionInfo
		ioHandler      client.IOHandler
		done           chan struct{}
		lock           sync.RWMutex
		ns             string
		pending        map[string]chan client.EvalResult
		inputRequested bool
		inputBuffer    *strings.Builder
	}

	Opts struct {
		Host      string
		Port      int
		Oneshot   bool
		IOHandler client.IOHandler
	}
)

func NewClient(clientOpts *Opts) (*Client, error) {
	c := &Client{
		ns:        "user",
		ioHandler: clientOpts.IOHandler,
		done:      make(chan struct{}),
		pending:   map[string]chan client.EvalResult{},
	}
	opts := &ConnOpts{
		Host: clientOpts.Host,
		Port: clientOpts.Port,
	}
	conn, err := Connect(opts)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	if !clientOpts.Oneshot {
		sessionInfo, err := conn.initSession()
		if err != nil {
			return nil, err
		}
		c.sessionInfo = sessionInfo
	}
	go client.StartLoop(c.conn, c, c.done)
	return c, nil
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

func (c *Client) HandleResp(response client.Response) {
	//fmt.Printf("RESP: %v\n", resp)
	resp := response.(Response)
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
		ch <- client.NewRuntimeError(resp["ex"].(string))
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

func (c *Client) HandleErr(err error) {
	c.ioHandler.Err(err.Error(), true)
}

func (c *Client) handleStatusUpdate(resp Response) {
	status := resp["status"]
	if c.statusContains(status, "need-input") {
		c.lock.Lock()
		if buf := c.inputBuffer; buf != nil {
			in := buf.String()
			c.inputBuffer = nil
			c.lock.Unlock()
			c.sendStdin(in)
		} else {
			c.inputRequested = true
			c.lock.Unlock()
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
	if err := c.conn.Send(req); err != nil {
		c.ioHandler.Err(err.Error(), true)
	}
}

func (c *Client) newIdChan() (string, chan client.EvalResult) {
	id := uuid.NewString()
	ch := make(chan client.EvalResult)
	c.lock.Lock()
	c.pending[id] = ch
	c.lock.Unlock()
	return id, ch
}

func (c *Client) Eval(code string) <-chan client.EvalResult {
	id, ch := c.newIdChan()
	c.send(Request{
		"op":   "eval",
		"id":   id,
		"code": code,
		"ns":   c.CurrentNS(),
	})
	return ch
}

func (c *Client) Load(filename string, content string) <-chan client.EvalResult {
	id, ch := c.newIdChan()
	req := Request{
		"op":   "load-file",
		"id":   id,
		"ns":   c.CurrentNS(),
		"file": content,
	}
	if filename != "-" {
		req["file-name"] = filepath.Base(filename)
		req["file-path"] = filepath.Dir(filename)
	}
	c.send(req)
	return ch
}

func (c *Client) sendStdin(in string) {
	c.send(Request{
		"op":    "stdin",
		"stdin": in,
	})
}

func (c *Client) Stdin(input string) {
	c.lock.Lock()
	if c.inputBuffer == nil {
		c.inputBuffer = new(strings.Builder)
	}
	c.inputBuffer.WriteString(input)
	if c.inputRequested {
		in := c.inputBuffer.String()
		c.inputBuffer = nil
		c.inputRequested = false
		c.lock.Unlock()
		c.sendStdin(in)
	} else {
		c.lock.Unlock()
	}
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
