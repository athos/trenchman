package nrepl

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/athos/trenchman/trenchman/bencode"
)

type (
	Request  map[string]bencode.Datum
	Response map[string]bencode.Datum

	Handler    func(Response)
	ErrHandler func(error)

	Conn struct {
		socket     net.Conn
		encoder    *bencode.Encoder
		decoder    *bencode.Decoder
		handler    Handler
		errHandler ErrHandler
	}

	ConnBuilder struct {
		Host       string
		Port       int
		Handler    Handler
		ErrHandler ErrHandler
	}
)

func NewBuilder(host string, port int) *ConnBuilder {
	return &ConnBuilder{Host: host, Port: port}
}

func (b *ConnBuilder) Connect() (conn *Conn, err error) {
	socket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", b.Host, b.Port))
	if err != nil {
		return
	}
	return &Conn{
		socket:     socket,
		encoder:    bencode.NewEncoder(socket),
		decoder:    bencode.NewDecoder(socket),
		handler:    b.Handler,
		errHandler: b.ErrHandler,
	}, nil
}

func (conn *Conn) sendReq(req Request) error {
	return conn.encoder.Encode(map[string]bencode.Datum(req))
}

func (conn *Conn) recvResp() (resp Response, err error) {
	datum, err := conn.decoder.Decode()
	if err != nil {
		return
	}
	dict, ok := datum.(map[string]bencode.Datum)
	if !ok {
		return nil, errors.New("response must be a dictionary")
	}
	return Response(dict), nil
}

func (conn *Conn) initSession() (session string, err error) {
	req := Request{
		"op": "clone",
		"id": "init",
	}
	if err = conn.sendReq(req); err != nil {
		return
	}
	resp, err := conn.recvResp()
	if err != nil {
		return
	}
	session, ok := resp["new-session"].(string)
	if !ok {
		err = fmt.Errorf("illegal session id: %v", resp["new-session"])
		return
	}
	return
}

func (conn *Conn) startLoop(ctx context.Context) {
	handler := conn.handler
	if handler == nil {
		handler = func(_ Response) {}
	}
	errHandler := conn.errHandler
	if errHandler == nil {
		errHandler = func(_ error) {}
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := conn.recvResp()
			if err != nil {
				errHandler(err)
				return
			}
			handler(resp)
		}
	}
}

func (conn *Conn) Close() error {
	return conn.socket.Close()
}
