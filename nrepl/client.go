package nrepl

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/athos/trenchman/bencode"
	"github.com/athos/trenchman/client"
	"github.com/google/uuid"
)

type (
	Client struct {
		conn           *Conn
		sessionInfo    *SessionInfo
		outputHandler  client.OutputHandler
		errHandler     client.ErrorHandler
		idGenerator    func() string
		done           chan struct{}
		lock           sync.RWMutex
		ns             string
		pending        map[string]chan client.EvalResult
		inputRequested bool
		inputBuffer    *strings.Builder
	}

	Opts struct {
		InitNS        string
		Oneshot       bool
		OutputHandler client.OutputHandler
		ErrorHandler  client.ErrorHandler
		ConnBuilder   client.ConnBuilder
		Debug         bool
		idGenerator   func() string
	}
)

func NewClient(opts *Opts) (*Client, error) {
	initNS := opts.InitNS
	if initNS == "" {
		initNS = "user"
	}
	c := &Client{
		outputHandler: opts.OutputHandler,
		errHandler:    opts.ErrorHandler,
		ns:            initNS,
		done:          make(chan struct{}),
		pending:       map[string]chan client.EvalResult{},
		idGenerator:   opts.idGenerator,
	}
	conn, err := Connect(&ConnOpts{opts.ConnBuilder, opts.Debug, c})
	if err != nil {
		return nil, err
	}
	c.conn = conn
	if !opts.Oneshot {
		sessionInfo, err := conn.initSession()
		if err != nil {
			return nil, err
		}
		c.sessionInfo = sessionInfo
	}
	if c.idGenerator == nil {
		c.idGenerator = uuid.NewString
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

func (c *Client) HandleDebugMessage(s string) {
	c.outputHandler.Debug(s)
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
		err := fmt.Errorf("unknown status returned: %v", statuses)
		c.HandleErr(err)
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
		c.outputHandler.Out(resp["out"].(string))
	case has(resp, "err"):
		c.outputHandler.Err(resp["err"].(string))
	// default:
	// 	c.HandleErr(fmt.Errorf("unknown response returned: %v", resp))
	}
	if has(resp, "status") {
		c.handleStatusUpdate(resp)
	}
}

func (c *Client) HandleErr(err error) {
	c.errHandler.HandleErr(err)
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
		c.HandleErr(err)
	}
}

func (c *Client) newIdChan() (string, chan client.EvalResult) {
	id := c.idGenerator()
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
