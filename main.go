package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/athos/trenchman/trenchman/nrepl"
)

func initConn(ch chan string, host string, port int) (*nrepl.Conn, error) {
	builder := nrepl.NewBuilder(host, port)
	builder.Handler = func(resp nrepl.Response) {
		if val, ok := resp["value"]; ok {
			ch <- val.(string)
		} else if status, ok := resp["status"]; !ok || status == nil {
			fmt.Fprintf(os.Stderr, "Unknown response returned: %v\n", resp)
		}
	}
	builder.ErrHandler = func(e error) {
		fmt.Fprintln(os.Stderr, e.Error())
	}
	return builder.Connect()
}

func startRepl(conn *nrepl.Conn, ch chan string) {
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		code, err := in.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			panic(err)
		}
		req := nrepl.Request{
			"op":   "eval",
			"code": code,
		}
		conn.Send(req)
		res := <-ch
		fmt.Println(res)
	}
}

func main() {
	ch := make(chan string)
	conn, err := initConn(ch, "127.0.0.1", 49913)
	if err != nil {
		panic(err)
	}
	startRepl(conn, ch)
}
