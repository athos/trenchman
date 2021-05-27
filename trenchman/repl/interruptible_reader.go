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
		notifyCh chan struct{}
		resultCh chan result
	}

	result struct {
		s   string
		err error
	}
)

var errInterrupted = errors.New("read interrupted")

func newReader(cancelCh chan struct{}, r io.Reader) *interruptibleReader {
	reader := &interruptibleReader{
		reader:   bufio.NewReader(r),
		cancelCh: cancelCh,
		notifyCh: make(chan struct{}),
		resultCh: make(chan result),
	}
	go func() {
		for range reader.notifyCh {
			var res result
			res.s, res.err = reader.reader.ReadString('\n')
			reader.resultCh <- res
		}
	}()
	return reader
}

func (r *interruptibleReader) ReadLine() (string, error) {
	select {
	case <-r.cancelCh:
		return "", errInterrupted
	case res := <-r.resultCh:
		return res.s, res.err
	case r.notifyCh <- struct{}{}:
		select {
		case <-r.cancelCh:
			return "", errInterrupted
		case res := <-r.resultCh:
			return res.s, res.err
		}
	}
}

func (r *interruptibleReader) Close() error {
	close(r.notifyCh)
	return nil
}
