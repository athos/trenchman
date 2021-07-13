package prepl

import (
	"net"
	"testing"

	"github.com/athos/trenchman/client"
	"github.com/stretchr/testify/assert"
)

func setupMock(steps []client.Step) *client.MockServer {
	s := make([]client.Step, 1, len(steps)+1)
	s[0] = client.Step{
		Expected:  "(set! *print-namespace-maps* false)",
		Responses: []string{`{:tag :ret, :val "nil"}`},
	}
	s = append(s, steps...)
	return client.NewMockServer(s)
}

func setupClient(mock *client.MockServer) (*Client, error) {
	return NewClient(&Opts{
		connBuilder: func(_ string, _ int) (net.Conn, error) {
			return mock, nil
		},
		OutputHandler: mock,
		ErrorHandler:  mock,
	})
}

func TestEval(t *testing.T) {
	tests := []struct {
		input  string
		step   client.Step
		ns     string
		result client.EvalResult
		outs   []string
		errs   []string
	}{
		{
			"(+ 1 2)",
			client.Step{
				Expected:  "(do (+ 1 2))",
				Responses: []string{`{:tag :ret, :val "3", :ns "user"}`},
			},
			"user",
			"3",
			nil,
			nil,
		},
		{
			"(ns foo)",
			client.Step{
				Expected:  "(do (ns foo))",
				Responses: []string{`{:tag :ret, :val "nil", :ns "foo"}`},
			},
			"foo",
			"nil",
			nil,
			nil,
		},
		{
			"(/ 1 0)",
			client.Step{
				Expected: "(do (/ 1 0))",
				Responses: []string{
					`{:tag :ret, :val "{:phase :execution, :cause \"Divide by zero\", :trace [[clojure.lang.Numbers divide \"Numbers.java\" 188]], :via [{:type java.lang.ArithmeticException, :message \"Divide by zero\", :at [clojure.lang.Numbers divide \"Numbers.java\" 188]}]}", :exception true, :ns "user"}`,
				},
			},
			"user",
			client.NewRuntimeError("Execution error (ArithmeticException) at clojure.lang.Numbers/divide (Numbers.java:188).\nDivide by zero"),
			nil,
			[]string{"Execution error (ArithmeticException) at clojure.lang.Numbers/divide (Numbers.java:188).\nDivide by zero\n"},
		},
		{
			"(run! prn (range 3))",
			client.Step{
				Expected: "(do (run! prn (range 3)))",
				Responses: []string{
					"{:tag :out, :val \"0\n\"}",
					"{:tag :out, :val \"1\n\"}",
					"{:tag :out, :val \"2\n\"}",
					`{:tag :ret, :val "nil", :ns "user"}`,
				},
			},
			"user",
			"nil",
			[]string{"0\n", "1\n", "2\n"},
			nil,
		},
		{
			"(binding [*out* *err*] (prn 42))",
			client.Step{
				Expected: "(do (binding [*out* *err*] (prn 42)))",
				Responses: []string{
					"{:tag :err, :val \"42\n\"}",
					`{:tag :ret, :val "nil", :ns "user"}`,
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
			mock := setupMock([]client.Step{tt.step})
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
	t.Run("(read-line)", func(t *testing.T) {
		steps := []client.Step{
			{
				Expected:  "(do (read-line))",
				Responses: nil,
			},
			{
				Expected: "foo\n",
				Responses: []string{
					"{:tag :ret, :val \"\\\"foo\\\"\\n\", :ns \"user\"}",
				},
			},
		}
		mock := setupMock(steps)
		c, err := setupClient(mock)
		assert.Nil(t, err)
		ch := c.Eval("(read-line)")
		go func() {
			c.Stdin("foo\n")
		}()
		ret := <-ch
		assert.Equal(t, "\"foo\"\n", ret)
		assert.Nil(t, mock.HandledErr())
		assert.Nil(t, mock.Outs())
		assert.Nil(t, mock.Errs())
		assert.Nil(t, c.Close())
	})
}

func TestLoad(t *testing.T) {
	mock := setupMock([]client.Step{
		{
			Expected: "(do (println \"Hello, World!\"))",
			Responses: []string{
				"{:tag :out, :val \"Hello, World!\n\"}",
				`{:tag :ret, :val "nil"}`,
			},
		},
	})
	c, err := setupClient(mock)
	assert.Nil(t, err)
	ch := c.Load("hello.clj", "(println \"Hello, World!\")")
	ret := <-ch
	assert.Equal(t, "nil", ret)
	assert.Nil(t, mock.HandledErr())
	assert.Equal(t, []string{"Hello, World!\n"}, mock.Outs())
	assert.Nil(t, mock.Errs())
	assert.Nil(t, c.Close())
}
