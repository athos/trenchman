package nrepl

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/athos/trenchman/bencode"
	"github.com/athos/trenchman/client"
)

type (
	Request  map[string]bencode.Datum
	Response map[string]bencode.Datum

	Handler    func(Response)
	ErrHandler func(error)

	Conn struct {
		socket  net.Conn
		encoder *bencode.Encoder
		decoder *bencode.Decoder
	}

	SessionInfo struct {
		session string
		ops     map[string]struct{}
	}
)

func Connect(host string, port int) (conn *Conn, err error) {
	socket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return
	}
	return &Conn{
		socket:  socket,
		encoder: bencode.NewEncoder(socket),
		decoder: bencode.NewDecoder(socket),
	}, nil
}

func (conn *Conn) Send(req client.Request) error {
	return conn.encoder.Encode(map[string]bencode.Datum(req.(Request)))
}

func (conn *Conn) Recv() (client.Response, error) {
	datum, err := conn.decoder.Decode()
	if err != nil {
		if err == io.EOF {
			err = client.ErrDisconnected
		}
		return nil, err
	}
	dict, ok := datum.(map[string]bencode.Datum)
	if !ok {
		return nil, errors.New("response must be a dictionary")
	}
	return Response(dict), nil
}

func (conn *Conn) initSession() (ret *SessionInfo, err error) {
	req := Request{
		"op": "clone",
		"id": "init",
	}
	if err = conn.Send(req); err != nil {
		return
	}
	response, err := conn.Recv()
	if err != nil {
		return
	}
	resp := response.(Response)
	session, ok := resp["new-session"].(string)
	if !ok {
		err = fmt.Errorf("illegal session id: %v", resp["new-session"])
		return
	}
	if err = conn.Send(Request{"op": "describe"}); err != nil {
		return
	}
	response, err = conn.Recv()
	if err != nil {
		return
	}
	resp = response.(Response)
	ops := map[string]struct{}{}
	for k, _ := range resp["ops"].(map[string]bencode.Datum) {
		ops[k] = struct{}{}
	}
	ret = &SessionInfo{
		session: session,
		ops:     ops,
	}
	return
}

func (conn *Conn) Close() error {
	return conn.socket.Close()
}
