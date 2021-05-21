package bencode

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		in Datum
		out string
	}{
		{42, "i42e"},
		{-42, "i-42e"},
		{"foobar", "6:foobar"},
		{[]Datum{"foo", "bar", "baz"}, "l3:foo3:bar3:baze"},
		{
			map[string]Datum{
				"foo": 100,
				"bar": 200,
				"baz": 300,
			},
			"d3:bari200e3:bazi300e3:fooi100ee",
		},
		{
			[]Datum{
				map[string]Datum{"name": "alice", "age": 20},
				map[string]Datum{"name": "bob", "age": 30},
			},
			"ld3:agei20e4:name5:aliceed3:agei30e4:name3:bobee",
		},
	}
	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			var b strings.Builder
			err := Encode(&b, tt.in)
			assert.Equal(t, tt.out, b.String())
			assert.Nil(t, err)
		})
	}
}
