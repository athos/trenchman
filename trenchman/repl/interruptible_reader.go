package repl

import (
	"bufio"
	"errors"
	"io"
)

type (
	interruptibleReader struct {
		reader   *bufio.Reader
		cancelCh chan struct{}
		notifyCh chan chan result
	}

	result struct {
		s   string
		err error
	}
)

var errInterrupted = errors.New("read interrupted")

func newReader(cancelCh chan struct{}, r io.Reader) *interruptibleReader {
	notifyCh := make(chan chan result)
	reader := &interruptibleReader{
		reader:   bufio.NewReader(r),
		cancelCh: cancelCh,
		notifyCh: notifyCh,
	}
	go func() {
		for c := range notifyCh {
			var res result
			res.s, res.err = reader.reader.ReadString('\n')
			select {
			case c <- res:
			default:
			}
		}
	}()
	return reader
}

func (r *interruptibleReader) ReadLine() (string, error) {
	resultCh := make(chan result)
	r.notifyCh <- resultCh
	select {
	case res := <-resultCh:
		return res.s, res.err
	case <-r.cancelCh:
		return "", errInterrupted
	}
}

func (r *interruptibleReader) Dispose() {
	close(r.notifyCh)
}
