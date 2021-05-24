package nrepl

import (
	"fmt"
)

type IOHandler interface {
	In() string
	Out(s string)
	Err(s string, fatal bool)
}

type Client struct {
	conn      *Conn
	session   string
	ch        chan string
	ioHandler IOHandler
	done      chan struct{}
}

func NewClient(host string, port int, ioHandler IOHandler) (*Client, error) {
	ch := make(chan string)
	client := &Client{
		ch:        ch,
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

func (c *Client) handleResp(resp Response) {
	if val, ok := resp["value"]; ok {
		c.ch <- val.(string)
	} else if status, ok := resp["status"]; !ok || status == nil {
		msg := fmt.Sprintf("Unknown response returned: %v", resp)
		c.ioHandler.Err(msg, true)
	}
}

func (c *Client) Eval(code string) string {
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
