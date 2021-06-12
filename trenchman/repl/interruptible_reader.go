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
		resultCh chan interface{}
		returnCh chan interface{}
	}
)

var errInterrupted = errors.New("read interrupted")

func newReader(r io.Reader) *interruptibleReader {
	reader := &interruptibleReader{
		reader:   bufio.NewReader(r),
		cancelCh: make(chan struct{}, 1),
		notifyCh: make(chan struct{}),
		resultCh: make(chan interface{}),
		returnCh: make(chan interface{}),
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

func (r *interruptibleReader) readLine() <-chan interface{} {
	go func() {
		//FIXME: added to ignore occasional panic that says "send on closed channel"
		defer func() {
			recover()
		}()
		select {
		case _, ok := <-r.cancelCh:
			if ok {
				r.returnCh <- errInterrupted
			}
		case res := <-r.resultCh:
			r.returnCh <- res
		case r.notifyCh <- struct{}{}:
			select {
			case <-r.cancelCh:
				r.returnCh <- errInterrupted
			case res := <-r.resultCh:
				r.returnCh <- res
			}
		}
	}()
	return r.returnCh
}

func (r *interruptibleReader) Close() error {
	close(r.notifyCh)
	return nil
}

func (r *interruptibleReader) interrupt() {
	r.cancelCh <- struct{}{}
}
