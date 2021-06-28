package nrepl

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/athos/trenchman/bencode"
	"github.com/athos/trenchman/client"
	"github.com/stretchr/testify/assert"
)

const (
	SESSION_ID = "1234"
	EXEC_ID    = "12345"
)

type step struct {
	expected  map[string]bencode.Datum
	responses []map[string]bencode.Datum
}

func encode(datum bencode.Datum) string {
	sb := new(strings.Builder)
	if err := bencode.Encode(sb, datum); err != nil {
		panic(fmt.Errorf("bencode encoding failed: %q", err))
	}
	return sb.String()
}

func setupMock(steps []step, autoIdEnabled bool) *client.MockServer {
	res := make([]client.Step, 2, len(steps)+1)
	res[0] = client.Step{
		Expected: encode(map[string]bencode.Datum{
			"op": "clone",
			"id": "init",
		}),
		Responses: []string{
			encode(map[string]bencode.Datum{
				"new-session": SESSION_ID,
			}),
		},
	}
	res[1] = client.Step{
		Expected: encode(map[string]bencode.Datum{
			"op": "describe",
		}),
		Responses: []string{
			encode(map[string]bencode.Datum{
				"ops": map[string]bencode.Datum{
					"eval":      true,
					"load-file": true,
				},
			}),
		},
	}
	for _, step := range steps {
		if autoIdEnabled {
			step.expected["session"] = SESSION_ID
			step.expected["id"] = EXEC_ID
		}
		s := client.Step{Expected: encode(step.expected)}
		for _, r := range step.responses {
			if autoIdEnabled {
				r["session"] = SESSION_ID
				r["id"] = EXEC_ID
			}
			s.Responses = append(s.Responses, encode(r))
		}
		res = append(res, s)
	}
	return client.NewMockServer(res)
}

func setupClient(mock *client.MockServer) (*Client, error) {
	return NewClient(&Opts{
		OutputHandler: mock,
		ErrorHandler:  mock,
		connBuilder: func(_ string, _ int) (net.Conn, error) {
			return mock, nil
		},
		idGenerator: func() string { return EXEC_ID },
	})
}

func TestEval(t *testing.T) {
	tests := []struct {
		input  string
		step   step
		ns     string
		result client.EvalResult
		outs   []string
		errs   []string
	}{
		{
			"(+ 1 2)",
			step{
				expected: map[string]bencode.Datum{
					"op":   "eval",
					"code": "(+ 1 2)",
					"ns":   "user",
				},
				responses: []map[string]bencode.Datum{
					{"ns": "user", "value": "3"},
					{"status": []bencode.Datum{"done"}},
				},
			},
			"user",
			"3",
			nil,
			nil,
		},
		{
			"(ns foo)",
			step{
				expected: map[string]bencode.Datum{
					"op":   "eval",
					"code": "(ns foo)",
					"ns":   "user",
				},
				responses: []map[string]bencode.Datum{
					{"ns": "foo", "value": "nil"},
					{"status": []bencode.Datum{"done"}},
				},
			},
			"foo",
			"nil",
			nil,
			nil,
		},
		{
			"(/ 1 0)",
			step{
				expected: map[string]bencode.Datum{
					"op":   "eval",
					"code": "(/ 1 0)",
					"ns":   "user",
				},
				responses: []map[string]bencode.Datum{
					{"err": "Divide by zero\n"},
					{
						"ex":     "class java.lang.ArithmeticException",
						"status": []bencode.Datum{"eval-error"},
					},
					{"status": []bencode.Datum{"done"}},
				},
			},
			"user",
			client.NewRuntimeError("class java.lang.ArithmeticException"),
			nil,
			[]string{"Divide by zero\n"},
		},
		{
			"(run! prn (range 3))",
			step{
				expected: map[string]bencode.Datum{
					"op":   "eval",
					"code": "(run! prn (range 3))",
					"ns":   "user",
				},
				responses: []map[string]bencode.Datum{
					{"out": "0\n"},
					{"out": "1\n"},
					{"out": "2\n"},
					{"ns": "user", "value": "nil"},
					{"status": []bencode.Datum{"done"}},
				},
			},
			"user",
			"nil",
			[]string{"0\n", "1\n", "2\n"},
			nil,
		},
		{
			"(binding [*out* *err*] (prn 42))",
			step{
				expected: map[string]bencode.Datum{
					"op":   "eval",
					"code": "(binding [*out* *err*] (prn 42))",
					"ns":   "user",
				},
				responses: []map[string]bencode.Datum{
					{"err": "42\n"},
					{"ns": "user", "value": "nil"},
					{"status": []bencode.Datum{"done"}},
				},
			},
			"user",
			"nil",
			nil,
			[]string{"42\n"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mock := setupMock([]step{tt.step}, true)
			c, err := setupClient(mock)
			assert.Nil(t, err)
			ch := c.Eval(tt.input)
			ret := <-ch
			assert.Equal(t, tt.result, ret)
			assert.Equal(t, tt.ns, c.CurrentNS())
			assert.Nil(t, mock.HandledErr())
			assert.Equal(t, tt.outs, mock.Outs())
			assert.Equal(t, tt.errs, mock.Errs())
			assert.Nil(t, c.Close())
		})
	}
}

func TestStdin(t *testing.T) {
	tests := []struct {
		title string
		sleep bool
	}{
		{"need-input arrives after stdin", false},
		{"need-input arrives before stdin", true},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			steps := []step{
				{
					expected: map[string]bencode.Datum{
						"session": SESSION_ID,
						"id":      EXEC_ID,
						"op":      "eval",
						"code":    "(read-line)",
						"ns":      "user",
					},
					responses: []map[string]bencode.Datum{
						{
							"session": SESSION_ID,
							"id":      EXEC_ID,
							"status":  []bencode.Datum{"need-input"},
						},
					},
				},
				{
					expected: map[string]bencode.Datum{
						"session": SESSION_ID,
						"op":      "stdin",
						"stdin":   "foo\n",
					},
					responses: []map[string]bencode.Datum{
						{
							"session": SESSION_ID,
							"status":  []bencode.Datum{"done"},
						},
						{
							"session": SESSION_ID,
							"id":      EXEC_ID,
							"ns":      "user",
							"value":   "\"foo\"",
						},
						{
							"session": SESSION_ID,
							"id":      EXEC_ID,
							"status":  []bencode.Datum{"done"},
						},
					},
				},
			}
			mock := setupMock(steps, false)
			c, err := setupClient(mock)
			assert.Nil(t, err)
			ch := c.Eval("(read-line)")
			go func() {
				if tt.sleep {
					time.Sleep(50 * time.Millisecond)
				}
				c.Stdin("foo\n")
			}()
			ret := <-ch
			assert.Equal(t, "\"foo\"", ret)
			assert.Equal(t, "user", c.CurrentNS())
			assert.Nil(t, mock.HandledErr())
			assert.Nil(t, mock.Outs())
			assert.Nil(t, mock.Errs())
			assert.Nil(t, c.Close())
		})
	}
}
