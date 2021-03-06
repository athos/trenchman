package client

import (
	"fmt"
	"net"
	"time"
)

type ConnBuilder interface {
	Connect() (net.Conn, error)
}

type TCPConnBuilder struct {
	Host string
	Port int
}

type UnixConnBuilder struct {
	Path string
}

func (builder *TCPConnBuilder) Connect() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", builder.Host, builder.Port))
}

func (builder *UnixConnBuilder) Connect() (net.Conn, error) {
	return net.Dial("unix", builder.Path)
}

type ConnBuilderFunc func() (net.Conn, error)

func (f ConnBuilderFunc) Connect() (net.Conn, error) {
	return f()
}

func NewRetryConnBuilder(connBuilder ConnBuilder, timeout time.Duration, interval time.Duration) ConnBuilder {
	if interval > timeout {
		interval = timeout
	}
	return ConnBuilderFunc(func() (conn net.Conn, err error) {
		end := time.Now().Add(timeout)
		for {
			conn, err = connBuilder.Connect()
			if err == nil {
				return conn, nil
			}
			if time.Now().After(end) {
				return
			}
			time.Sleep(interval)
			if time.Now().After(end) {
				return
			}
		}
	})
}
