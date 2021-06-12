package prepl

import (
	"bufio"
	"fmt"
	"net"
	"sync"

	"github.com/athos/trenchman/trenchman/client"
	"olympos.io/encoding/edn"
)

type (
	Response struct {
		Tag       edn.Keyword
		Val       string
		Ns        string
		Ms        int
		Form      string
		Exception bool
	}

	Client struct {
		socket        net.Conn
		decoder       *edn.Decoder
		writer        *bufio.Writer
		outputHandler client.OutputHandler
		lock          sync.RWMutex
		ns            string
		returnCh      chan client.EvalResult
		done          chan struct{}
	}

	Opts struct {
		Host          string
		Port          int
		OutputHandler client.OutputHandler
	}
)

func NewClient(opts *Opts) (*Client, error) {
	socket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", opts.Host, opts.Port))
	if err != nil {
		return nil, err
	}
	c := &Client{
		socket:        socket,
		decoder:       edn.NewDecoder(socket),
		writer:        bufio.NewWriter(socket),
		outputHandler: opts.OutputHandler,
		ns:            "user",
		done:          make(chan struct{}),
	}
	if err := c.Send("(set! *print-namespace-maps* false)"); err != nil {
		return nil, err
	}
	if _, err := c.Recv(); err != nil {
		return nil, err
	}
	go client.StartLoop(c, c, c.done)
	return c, nil
}

func (c *Client) Close() error {
	close(c.done)
	return c.socket.Close()
}

func (c *Client) Send(code client.Request) error {
	if _, err := c.writer.WriteString(code.(string)); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) Recv() (client.Response, error) {
	var resp Response
	if err := c.decoder.Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) HandleResp(response client.Response) {
	resp := response.(*Response)
	// fmt.Printf("resp: %v\n", resp)
	switch resp.Tag.String() {
	case ":ret":
		c.handleResult(resp)
	case ":out":
		c.outputHandler.Out(resp.Val)
	case ":err":
		c.outputHandler.Err(resp.Val, false)
	case ":tap":
	default:
		panic(fmt.Errorf("unknown response: %v", resp.Tag))
	}
}

func (c *Client) handleResult(resp *Response) {
	c.lock.Lock()
	ch := c.returnCh
	c.returnCh = nil
	c.ns = resp.Ns
	c.lock.Unlock()
	if resp.Exception {
		c.outputHandler.Err(errorMessage(resp.Val)+"\n", false)
		ch <- client.NewRuntimeError(resp.Val)
	} else {
		ch <- resp.Val
	}
	close(ch)
}

func (c *Client) HandleErr(err error) {
	c.outputHandler.Err(err.Error(), true)
}

func (c *Client) CurrentNS() string {
	c.lock.RLock()
	ns := c.ns
	c.lock.RUnlock()
	return ns
}

func (c *Client) SupportsOp(op string) bool {
	return op != "load-file" && op != "interrupt"
}

func (c *Client) Eval(code string) <-chan client.EvalResult {
	ch := make(chan client.EvalResult)
	c.lock.Lock()
	c.returnCh = ch
	c.lock.Unlock()
	if err := c.Send(code + "\n"); err != nil {
		c.outputHandler.Err(err.Error(), true)
	}
	return ch
}

func (c *Client) Load(filename string, content string) <-chan client.EvalResult {
	panic("load not supported")
}

func (c *Client) Stdin(input string) {
	if err := c.Send(input); err != nil {
		c.HandleErr(err)
	}
}

func (c *Client) Interrupt() {
	panic("interrupt not supported")
}
