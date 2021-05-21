package bencode

import (
	"bufio"
	"fmt"
	"io"
)

type Datum interface{}

type Decoder struct {
	reader *bufio.Reader
}

func NewDecoder(reader io.Reader) *Decoder {
	return &Decoder{bufio.NewReader(reader)}
}

func (d *Decoder) readByte() (byte, error) {
	return d.reader.ReadByte()
}

func (d *Decoder) unreadByte() {
	if err := d.reader.UnreadByte(); err != nil {
		panic(err)
	}
}

func (d *Decoder) ensureByte(b byte, expected byte) error {
	if b != expected {
		return fmt.Errorf("'%c' expected, but got '%c'", expected, b)
	}
	return nil
}

func (d *Decoder) Decode() (Datum, error) {
	b, err := d.readByte()
	if err != nil {
		return nil, err
	}
	switch b {
	case 'i':
		return d.decodeInt()
	case 'l':
		return d.decodeList()
	case 'd':
		return d.decodeDict()
	default:
		if '0' <= b && b <= '9' || b == '-' {
			d.unreadByte()
			return d.decodeString()
		}
	}
	return nil, nil
}

func (d *Decoder) decodeNumber(delim byte) (n int, err error) {
	for {
		b, err := d.readByte()
		if err != nil {
			return 0, err
		}
		switch {
		case '0' <= b && b <= '9':
			n = n*10 + int(b-'0')
		default:
			if err := d.ensureByte(b, delim); err != nil {
				return 0, err
			}
			return n, nil
		}
	}
}

func (d *Decoder) decodeInt() (Datum, error) {
	b, _ := d.readByte()
	negative := false
	if b == '-' {
		negative = true
	} else {
		d.unreadByte()
	}
	n, err := d.decodeNumber('e')
	if err != nil {
		return nil, err
	}
	if negative {
		n = -n
	}
	return n, nil
}

func (d *Decoder) decodeString() (ret string, err error) {
	n, err := d.decodeNumber(':')
	if err != nil {
		return
	}
	bs := make([]byte, n)
	m, err := d.reader.Read(bs)
	if err != nil {
		return
	}
	if m != n {
		return ret, io.ErrUnexpectedEOF
	}
	return string(bs), nil
}

func (d *Decoder) decodeList() (Datum, error) {
	elems := []Datum{}
	for {
		b, err := d.readByte()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			return elems, nil
		}
		d.unreadByte()
		elem, err := d.Decode()
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
}

func (d *Decoder) decodeDict() (Datum, error) {
	elems := map[string]Datum{}
	for {
		b, err := d.readByte()
		if err != nil {
			return nil, err
		}
		if b == 'e' {
			return elems, nil
		}
		d.unreadByte()
		k, err := d.decodeString()
		if err != nil {
			return nil, err
		}
		v, err := d.Decode()
		if err != nil {
			return nil, err
		}
		elems[k] = v
	}
}

func Decode(reader io.Reader) (Datum, error) {
	d := NewDecoder(reader)
	return d.Decode()
}
