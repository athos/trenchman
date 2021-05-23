package nrepl

import (
	"errors"
	"fmt"
	"net"

	"github.com/athos/trenchman/trenchman/bencode"
)

type (
	Request  map[string]bencode.Datum
	Response map[string]bencode.Datum
	Conn     struct {
		socket     net.Conn
		encoder    *bencode.Encoder
		decoder    *bencode.Decoder
		session    string
		errHandler ErrHandler
	}

	Handler     func(Response)
	ErrHandler  func(error)
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
	conn = &Conn{
		socket:     socket,
		encoder:    bencode.NewEncoder(socket),
		decoder:    bencode.NewDecoder(socket),
		errHandler: b.ErrHandler,
	}
	if err = conn.initSession(); err != nil {
		return
	}
	go conn.startLoop(b.Handler)
	return
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

func (conn *Conn) initSession() error {
	req := Request{
		"op": "clone",
		"id": "init",
	}
	if err := conn.sendReq(req); err != nil {
		return err
	}
	resp, err := conn.recvResp()
	if err != nil {
		return err
	}
	session, ok := resp["new-session"].(string)
	if !ok {
		return fmt.Errorf("illegal session id: %v", resp["new-session"])
	}
	conn.session = session
	return nil
}

func (conn *Conn) startLoop(listener Handler) {
	for {
		resp, err := conn.recvResp()
		if err != nil && conn.errHandler != nil {
			conn.errHandler(err)
			break
		}
		listener(resp)
	}
}

func (conn *Conn) Close() error {
	return conn.socket.Close()
}

func (conn *Conn) Send(req Request) error {
	req["session"] = conn.session
	return conn.sendReq(req)
}