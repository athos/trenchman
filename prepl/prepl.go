package prepl

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/athos/trenchman/client"
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
		errHandler    client.ErrorHandler
		lock          sync.RWMutex
		ns            string
		returnCh      chan client.EvalResult
		done          chan struct{}
		debug         bool
	}

	Opts struct {
		InitNS        string
		OutputHandler client.OutputHandler
		ErrorHandler  client.ErrorHandler
		ConnBuilder   client.ConnBuilder
		Debug         bool
	}
)

func (resp *Response) String() string {
	bs, _ := edn.Marshal(resp)
	return string(bs)
}

func NewClient(opts *Opts) (*Client, error) {
	connBuilder := opts.ConnBuilder
	socket, err := connBuilder.Connect()
	if err != nil {
		return nil, err
	}
	initNS := opts.InitNS
	if initNS == "" {
		initNS = "user"
	}
	c := &Client{
		socket:        socket,
		decoder:       edn.NewDecoder(socket),
		writer:        bufio.NewWriter(socket),
		outputHandler: opts.OutputHandler,
		errHandler:    opts.ErrorHandler,
		ns:            initNS,
		done:          make(chan struct{}),
		debug:         opts.Debug,
	}
	if err := c.Send("(set! *print-namespace-maps* false)\n"); err != nil {
		return nil, err
	}
	if _, err := c.Recv(); err != nil {
		return nil, err
	}
	if initNS != "user" {
		msg := fmt.Sprintf("(require '%s)\n(in-ns '%s)\n", initNS, initNS)
		if err := c.Send(msg); err != nil {
			return nil, err
		}
		if _, err := c.Recv(); err != nil {
			return nil, err
		}
		if _, err := c.Recv(); err != nil {
			return nil, err
		}
	}
	go client.StartLoop(c, c, c.done)
	return c, nil
}

func (c *Client) Close() error {
	close(c.done)
	return c.socket.Close()
}

func (c *Client) Send(code client.Request) error {
	msg := code.(string)
	if c.debug {
		c.outputHandler.Debug(fmt.Sprintf("[DEBUG:SEND] %q\n", strings.TrimSpace(msg)))
	}
	if _, err := c.writer.WriteString(msg); err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) Recv() (client.Response, error) {
	var resp Response
	if err := c.decoder.Decode(&resp); err != nil {
		if err == io.EOF {
			err = client.ErrDisconnected
		}
		return nil, err
	}
	if c.debug {
		c.outputHandler.Debug(fmt.Sprintf("[DEBUG:RECV] %s\n", resp.String()))
	}
	return client.Response(&resp), nil
}

func (c *Client) HandleResp(response client.Response) {
	resp := response.(*Response)
	switch resp.Tag.String() {
	case ":ret":
		c.handleResult(resp)
	case ":out":
		c.outputHandler.Out(resp.Val)
	case ":err":
		c.outputHandler.Err(resp.Val)
	case ":tap":
	default:
		c.HandleErr(fmt.Errorf("unknown type of response received: %v", resp.Tag))
	}
}

func (c *Client) handleResult(resp *Response) {
	c.lock.Lock()
	ch := c.returnCh
	c.returnCh = nil
	c.ns = resp.Ns
	c.lock.Unlock()
	if resp.Exception {
		msg, err := errorMessage(resp.Val)
		if err != nil {
			msg = err.Error()
		}
		c.outputHandler.Err(msg + "\n")
		ch <- client.NewRuntimeError(msg)
	} else {
		ch <- resp.Val
	}
	close(ch)
}

func (c *Client) HandleErr(err error) {
	c.errHandler.HandleErr(err)
}

func (c *Client) CurrentNS() string {
	c.lock.RLock()
	ns := c.ns
	c.lock.RUnlock()
	return ns
}

func (c *Client) SupportsOp(op string) bool {
	switch op {
	case "eval", "load-file":
		return true
	default:
		return false
	}
}

func (c *Client) Eval(code string) <-chan client.EvalResult {
	ch := make(chan client.EvalResult)
	c.lock.Lock()
	c.returnCh = ch
	c.lock.Unlock()
	if err := c.Send(fmt.Sprintf("(do %s)", code)); err != nil {
		c.HandleErr(err)
	}
	return ch
}

func (c *Client) Load(filename string, content string) <-chan client.EvalResult {
	return c.Eval(content)
}

func (c *Client) Stdin(input string) {
	if err := c.Send(input); err != nil {
		c.HandleErr(err)
	}
}

func (c *Client) Interrupt() {
	panic("interrupt not supported")
}
