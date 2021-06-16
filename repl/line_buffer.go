package repl

import "fmt"

type lineBuffer struct {
	buf string
}

var pairSymbols = map[rune]rune{
	'(': ')',
	'{': '}',
	'[': ']',
}

func (b *lineBuffer) feedLine(line string) (ret string, continued bool, err error) {
	b.buf += line
	cs := []rune(b.buf)
	pending := []rune{}
	instr := false
	for i := 0; i < len(cs); i++ {
		c := cs[i]
		if !instr {
			switch c {
			case '(', '{', '[':
				pending = append(pending, pairSymbols[c])
			case ')', '}', ']':
				n := len(pending)
				if n == 0 || pending[n-1] != c {
					ret = b.buf
					b.buf = ""
					err = fmt.Errorf("unbalanced symbol found: %c", c)
					return
				}
				pending = pending[:n-1]
			case '"':
				instr = true
			case '\\':
				i++
			}
		} else {
			switch c {
			case '"':
				instr = false
			case '\\':
				i++
			}
		}
	}
	if len(pending) > 0 || instr {
		continued = true
		return
	}
	ret = b.buf
	b.buf = ""
	return
}

func (b *lineBuffer) reset() {
	b.buf = ""
}
