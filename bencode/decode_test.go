package bencode

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		in  string
		out Datum
	}{
		{"i42e", 42},
		{"i-42e", -42},
		{"6:foobar", "foobar"},
		{"l3:foo3:bar3:baze", []Datum{"foo", "bar", "baz"}},
		{
			"d3:fooi100e3:bari200e3:bazi300ee",
			map[string]Datum{
				"foo": 100,
				"bar": 200,
				"baz": 300,
			},
		},
		{
			"ld4:name5:alice3:agei20eed4:name3:bob3:agei30eee",
			[]Datum{
				map[string]Datum{"name": "alice", "age": 20},
				map[string]Datum{"name": "bob", "age": 30},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			res, err := Decode(strings.NewReader(tt.in))
			assert.Equal(t, tt.out, res)
			assert.Nil(t, err)
		})
	}
}
