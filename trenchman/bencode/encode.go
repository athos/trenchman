package bencode

import (
	"bufio"
	"io"
	"sort"
	"strconv"
)

type Encoder struct {
	writer *bufio.Writer
}

func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{bufio.NewWriter(writer)}
}

func (e *Encoder) writeByte(b byte) error {
	return e.writer.WriteByte(b)
}

func (e *Encoder) writeString(s string) (err error) {
	_, err = e.writer.WriteString(s)
	return
}

func (e *Encoder) encode1(datum Datum) error {
	switch datum := datum.(type) {
	case int:
		e.writeByte('i')
		e.writeString(strconv.Itoa(datum))
		e.writeByte('e')
	case string:
		e.writeString(strconv.Itoa(len(datum)))
		e.writeByte(':')
		e.writeString(datum)
	case []Datum:
		e.writeByte('l')
		for _, d := range datum {
			if err := e.encode1(d); err != nil {
				return err
			}
		}
		e.writeByte('e')
	case map[string]Datum:
		e.writeByte('d')
		keys := make([]string, 0, len(datum))
		for k := range datum {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := datum[k]
			if err := e.encode1(k); err != nil {
				return err
			}
			if err := e.encode1(v); err != nil {
				return err
			}
		}
		e.writeByte('e')
	}
	return nil
}

func (e *Encoder) Encode(datum Datum) error {
	if err := e.encode1(datum); err != nil {
		return err
	}
	return e.writer.Flush()
}

func Encode(writer io.Writer, datum Datum) (err error) {
	e := NewEncoder(writer)
	return e.Encode(datum)
}
