package repl

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

type Printer interface {
	With(...color.Attribute) Printer
	Fprint(io.Writer, ...interface{}) (int, error)
	Fprintln(io.Writer, ...interface{}) (int, error)
	Fprintf(io.Writer, string, ...interface{}) (int, error)
}

type MonochromePrinter struct{}

func NewMonochromePrinter() MonochromePrinter {
	return MonochromePrinter{}
}

func (p MonochromePrinter) With(_ ...color.Attribute) Printer {
	return p
}

func (p MonochromePrinter) Fprint(w io.Writer, arg ...interface{}) (int, error) {
	return fmt.Fprint(w, arg...)
}

func (p MonochromePrinter) Fprintln(w io.Writer, arg ...interface{}) (int, error) {
	return fmt.Fprintln(w, arg...)
}

func (p MonochromePrinter) Fprintf(w io.Writer, s string, arg ...interface{}) (int, error) {
	return fmt.Fprintf(w, s, arg...)
}

type ColorPrinter struct {
	*color.Color
}

func NewColorPrinter() ColorPrinter {
	return ColorPrinter{color.New()}
}

func (p ColorPrinter) With(attr ...color.Attribute) Printer {
	return ColorPrinter{p.Add(attr...)}
}

func NewPrinter(colorEnabled bool) Printer {
	if colorEnabled {
		return NewColorPrinter()
	} else {
		return NewMonochromePrinter()
	}
}
