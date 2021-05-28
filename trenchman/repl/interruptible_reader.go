package repl

import (
	"bufio"
	"errors"
	"io"
)

type (
	interruptibleReader struct {
		reader   *bufio.Reader
		cancelCh <-chan struct{}
		notifyCh chan struct{}
		resultCh chan interface{}
	}
)

var errInterrupted = errors.New("read interrupted")

func newReader(cancelCh <-chan struct{}, r io.Reader) *interruptibleReader {
	reader := &interruptibleReader{
		reader:   bufio.NewReader(r),
		cancelCh: cancelCh,
		notifyCh: make(chan struct{}),
		resultCh: make(chan interface{}),
	}
	go func() {
		for range reader.notifyCh {
			if res, err := reader.reader.ReadString('\n'); err != nil {
				reader.resultCh <- err
			} else {
				reader.resultCh <- res
			}
		}
	}()
	return reader
}

func (r *interruptibleReader) readLine() (string, error) {
	select {
	case <-r.cancelCh:
		return "", errInterrupted
	case res := <-r.resultCh:
		if s, ok := res.(string); !ok {
			return "", res.(error)
		} else {
			return s, nil
		}
	case r.notifyCh <- struct{}{}:
		select {
		case <-r.cancelCh:
			return "", errInterrupted
		case res := <-r.resultCh:
			if s, ok := res.(string); !ok {
				return "", res.(error)
			} else {
				return s, nil
			}
		}
	}
}

func (r *interruptibleReader) Close() error {
	close(r.notifyCh)
	return nil
}
