package repl

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type threadSafeBuffer struct {
	buf  *bytes.Buffer
	lock sync.Mutex
}

func newThreadSafeBuffer() *threadSafeBuffer {
	return &threadSafeBuffer{buf: new(bytes.Buffer)}
}

func (b *threadSafeBuffer) Read(bytes []byte) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buf.Read(bytes)
}

func (b *threadSafeBuffer) WriteString(s string) (int, error) {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.buf.WriteString(s)
}

func TestReadLine(t *testing.T) {
	b := newThreadSafeBuffer()
	r := newReader(b)
	ch := r.readLine()
	go b.WriteString("hello\n")
	line := <-ch
	assert.Equal(t, "hello\n", line)
	assert.Nil(t, r.Close())
}

func TestInterrupt(t *testing.T) {
	t.Run("interrupt interrupts readLine", func(t *testing.T) {
		b := newThreadSafeBuffer()
		r := newReader(b)
		ch := r.readLine()
		go func() {
			r.interrupt()
		}()
		res := <-ch
		_, ok := res.(error)
		assert.True(t, ok)
		assert.Nil(t, r.Close())
	})
	// t.Run("readLine after interruption successfully reads line", func(t *testing.T) {
	// 	b := newThreadSafeBuffer()
	// 	r := newReader(b)
	// 	ch := r.readLine()
	// 	go func() {
	// 		r.interrupt()
	// 	}()
	// 	res := <-ch
	// 	_, ok := res.(error)
	// 	assert.True(t, ok)
	// 	ch = r.readLine()
	// 	go func() {
	// 		b.WriteString("hello\n")
	// 	}()
	// 	line := <-ch
	// 	assert.Equal(t, "hello\n", line)
	// 	assert.Nil(t, r.Close())
	// })
}
