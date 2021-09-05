package client

import (
	"fmt"
	"net"
)

type ConnBuilder interface {
	Connect() (net.Conn, error)
}

type TCPConnBuilder struct {
	Host string
	Port int
}

func (conn *TCPConnBuilder) Connect() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", conn.Host, conn.Port))
}

type ConnBuilderFunc func() (net.Conn, error)

func (f ConnBuilderFunc) Connect() (net.Conn, error) {
	return f()
}
