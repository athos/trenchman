package prepl

import (
	"fmt"
	"path/filepath"
	"strings"

	"olympos.io/encoding/edn"
)

type (
	Exception struct {
		Phase edn.Keyword
		Trace [][]interface{} // source, method, file, line
		Via   []ViaEntry
	}

	ViaEntry struct {
		Type    edn.Symbol
		Message string
		Data    map[edn.Keyword]interface{}
	}

	TriageData struct {
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

func exTriage(ex *Exception) *TriageData {
	phase := ex.Phase.String()[1:]
	if phase == "" {
		phase = "execution"
	}
	td := TriageData{phase: phase}
	via := ex.Via
	var typ, msg string
	var topData map[edn.Keyword]interface{}
	var source string
	if len(via) > 0 {
		lastEntry := via[len(via)-1]
		typ = lastEntry.Type.String()
		msg = lastEntry.Message
		topData = via[0].Data
		if src, ok := topData[edn.Keyword("clojure.error/source")]; ok {
			source = src.(string)
		}
	}
	switch phase {
	case "read-source":
		if source != "" &&
			source != "NO_SOURCE_FILE" &&
			source != "NO_SOURCE_PATH" {
			td.source = filepath.Base(source)
			td.path = filepath.Dir(source)
		}
		if msg != "" {
			td.cause = msg
		}
	case "compile-syntax-check", "compilation", "macro-syntax-check", "macroexpansion":
		mergeToTriageData(td, topData)
		if source != "" &&
			source != "NO_SOURCE_FILE" &&
			source != "NO_SOURCE_PATH" {
			td.source = filepath.Base(source)
			td.path = filepath.Dir(source)
		}
		if typ != "" {
			td.class = typ
		}
		if msg != "" {
			td.cause = msg
		}
	case "read-eval-result", "print-eval-result":
		mergeToTriageData(td, topData)
		if len(ex.Trace) > 0 {
			source, method, file, line := coerceTraceEntry(ex.Trace[0])
			if line != 0 {
				td.line = line
			}
			if file != "" {
				td.source = file
			}
			if source != "" && method != "" {
				td.symbol = javaLocToSource(source, method)
			}
		}
		if typ != "" {
			td.class = typ
		}
		if msg != "" {
			td.cause = msg
		}
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

func mergeToTriageData(td TriageData, data map[edn.Keyword]interface{}) {
	if data == nil {
		return
	}
	if phase, ok := data[edn.Keyword("clojure.error/phase")]; ok {
		td.phase = phase.(edn.Keyword).String()[1:]
	}
	if line, ok := data[edn.Keyword("clojure.error/line")]; ok {
		td.line = int(line.(int64))
	}
	if column, ok := data[edn.Keyword("clojure.error/column")]; ok {
		td.column = int(column.(int64))
	}
	if source, ok := data[edn.Keyword("clojure.error/source")]; ok {
		td.source = source.(string)
	}
	if symbol, ok := data[edn.Keyword("clojure.error/symbol")]; ok {
		td.symbol = symbol.(edn.Symbol).String()
	}
}

func coerceTraceEntry(entry []interface{}) (string, string, string, int) {
	source := entry[0].(edn.Symbol).String()
	method := entry[1].(edn.Symbol).String()
	file := entry[2].(string)
	line := int(entry[3].(int64))
	return source, method, file, line
}

func findFirstNonCoreEntry(trace [][]interface{}) (string, string, string, int, bool) {
	for _, entry := range trace {
		source, method, file, line := coerceTraceEntry(entry)
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

func exString(td *TriageData) string {
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
		return fmt.Sprintf(
			"Syntax error reading source at (%s).\n%s",
			loc, td.cause,
		)
	case "macro-syntax-check":
		symbol := td.symbol
		if symbol != "" {
			symbol += " "
		}
		return fmt.Sprintf(
			"Syntax error macroexpanding %sat (%s).\n%s",
			symbol, loc, td.cause,
		)
	case "macroexpansion":
		symbol := td.symbol
		if symbol != "" {
			symbol += " "
		}
		return fmt.Sprintf(
			"Unexpected error%s macroexpanding %sat (%s).\n%s",
			causeType, symbol, loc, td.cause,
		)
	case "compile-syntax-check":
		symbol := td.symbol
		if symbol != "" {
			symbol += " "
		}
		return fmt.Sprintf(
			"Syntax error%s compiling %sat (%s).\n%s",
			causeType, symbol, loc, td.cause,
		)
	case "compilation":
		symbol := td.symbol
		if symbol != "" {
			symbol += " "
		}
		return fmt.Sprintf(
			"Unexpected error%s compiling %sat (%s).\n%s",
			causeType, symbol, loc, td.cause,
		)
	case "read-eval-result":
		return fmt.Sprintf(
			"Error reading eval result%s at %s (%s).\n%s",
			causeType, td.symbol, loc, td.cause,
		)
	case "print-eval-result":
		return fmt.Sprintf(
			"Error printing return value%s at %s (%s).\n%s",
			causeType, td.symbol, loc, td.cause,
		)
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
	var ex Exception
	if err := edn.UnmarshalString(payload, &ex); err != nil {
		panic("failed to parse exception data: " + err.Error())
	}
	return exString(exTriage(&ex))
}
