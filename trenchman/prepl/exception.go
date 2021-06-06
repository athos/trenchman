package prepl

import (
	"fmt"
	"strings"

	"olympos.io/encoding/edn"
)

type (
	exception struct {
		Phase edn.Keyword
		Trace [][]interface{} // source, method, file, line
		Via   []viaEntry
	}

	viaEntry struct {
		Type    edn.Symbol
		Message string
		// Data    map[edn.Keyword]interface{}
	}

	triageData struct {
		phase  string
		source string
		path   string
		line   int
		column int
		symbol string
		class  string
		cause  string
	}
)

func exTriage(ex *exception) *triageData {
	phase := ex.Phase.String()[1:]
	if phase == "" {
		phase = "execution"
	}
	td := triageData{phase: phase}
	via := ex.Via
	var typ, msg string
	if len(via) > 0 {
		lastEntry := via[len(via)-1]
		typ = lastEntry.Type.String()
		msg = lastEntry.Message
	}
	switch phase {
	case "read-source":
		//TODO
	case "compile-syntax-check", "compilation", "macro-syntax-check", "macroexpansion":
		//TODO
	case "read-eval-result", "print-eval-result":
		//TODO
	case "execution":
		td.class = typ
		if msg != "" {
			td.cause = msg
		}
		source, method, file, line, found := findFirstNonCoreEntry(ex.Trace)
		if found {
			td.symbol = javaLocToSource(source, method)
			td.source = file
			td.line = line
		}
	}
	return &td
}

func findFirstNonCoreEntry(trace [][]interface{}) (string, string, string, int, bool) {
	for _, entry := range trace {
		source := entry[0].(edn.Symbol).String()
		method := entry[1].(edn.Symbol).String()
		file := entry[2].(string)
		line := int(entry[3].(int64))
		if isCoreClass(source) {
			return source, method, file, line, true
		}
	}
	return "", "", "", 0, false
}

func javaLocToSource(class, method string) string {
	return class + "/" + method
}

func isCoreClass(className string) bool {
	return strings.HasPrefix(className, "clojure.")
}

func exString(td *triageData) string {
	source := "REPL"
	if td.path != "" {
		source = td.path
	}
	if td.source != "" {
		source = td.source
	}
	line := td.line
	if line == 0 {
		line = 1
	}
	columnStr := ""
	if td.column > 0 {
		columnStr = fmt.Sprintf(":%d", td.column)
	}
	loc := fmt.Sprintf("%s:%d%s", source, line, columnStr)
	classNameComponents := strings.Split(td.class, ".")
	simpleClass := classNameComponents[len(classNameComponents)-1]
	var causeType string
	switch simpleClass {
	case "Exception", "RuntimeException":
		causeType = ""
	default:
		causeType = fmt.Sprintf(" (%s)", simpleClass)
	}
	switch td.phase {
	case "read-source":
		// TODO
	case "macro-syntax-check":
		// TODO
	case "macroexpansion":
		// TODO
	case "compile-syntax-check":
		// TODO
	case "compilation":
		// TODO
	case "read-eval-result":
		// TODO
	case "print-eval-result":
		// TODO
	case "execution":
		symbol := td.symbol
		if symbol != "" {
			symbol += " "
		}
		return fmt.Sprintf(
			"Execution error%s at %s(%s).\n%s",
			causeType, symbol, loc, td.cause,
		)
	}
	return ""
}

func errorMessage(payload string) string {
	var ex exception
	edn.UnmarshalString(payload, &ex)
	return exString(exTriage(&ex))
}
