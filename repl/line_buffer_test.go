package repl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineBuffer(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{
			[]string{"foo"},
			"foo",
		},
		{
			[]string{"(+ 1 2)"},
			"(+ 1 2)",
		},
		{
			[]string{"(+ (* 3 3) (* 4 4))"},
			"(+ (* 3 3) (* 4 4))",
		},
		{
			[]string{"[(f 1) (f 2)]"},
			"[(f 1) (f 2)]",
		},
		{
			[]string{"\"foo\""},
			"\"foo\"",
		},
		{
			[]string{"[\"foo\", \"bar\"]"},
			"[\"foo\", \"bar\"]",
		},
		{
			[]string{"\":-(\""},
			"\":-(\"",
		},
		{
			[]string{"\":-)\""},
			"\":-)\"",
		},
		{
			[]string{"\"foo\\\"bar\""},
			"\"foo\\\"bar\"",
		},
		{
			[]string{"[\\( \\)]"},
			"[\\( \\)]",
		},
		{
			[]string{
				"(defn fib [n]\n",
				"  (loop [n n, a 0, b 1]\n",
				"    (if (= n 0)\n",
				"      a\n",
				"      (recur (dec n) b (+ a b)))))\n",
			},
			`(defn fib [n]
  (loop [n n, a 0, b 1]
    (if (= n 0)
      a
      (recur (dec n) b (+ a b)))))
`,
		},
		{
			[]string{
				"\"foo\n",
				"bar\"\n",
			},
			"\"foo\nbar\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.input, ""), func(t *testing.T) {
			buf := new(lineBuffer)
			for _, line := range tt.input[:len(tt.input)-1] {
				_, continued, err := buf.feedLine(line)
				assert.True(t, continued)
				assert.Nil(t, err)
			}
			ret, continued, err := buf.feedLine(tt.input[len(tt.input)-1])
			assert.Equal(t, tt.expected, ret)
			assert.False(t, continued)
			assert.Nil(t, err)
		})
	}
	t.Run("unbalanced brackets raise an error", func(t *testing.T) {
		buf := new(lineBuffer)
		_, _, err := buf.feedLine(")")
		assert.NotNil(t, err)
	})
	t.Run("unmatched brackets raise an error", func(t *testing.T) {
		buf := new(lineBuffer)
		_, _, err := buf.feedLine("(]")
		assert.NotNil(t, err)
	})
}

func TestReset(t *testing.T) {
	t.Run("reset clears buffered content", func(t *testing.T) {
		buf := new(lineBuffer)
		buf.feedLine("(foo\n")
		buf.feedLine(" bar\n")
		assert.Equal(t, "(foo\n bar\n", buf.buf)
		buf.reset()
		assert.Equal(t, "", buf.buf)
	})
}
