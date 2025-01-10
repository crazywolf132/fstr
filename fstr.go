package fstr

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Sprintf formats according to a format specifier (with Rust-like placeholders).
// See the doc comment for full details on placeholders, escaping, etc.
func Sprintf(format string, args ...interface{}) string {
	segments, placeholders := parseFormat(format)

	placeholderValues := make([]interface{}, len(placeholders))
	autoIndex := 0

	for i, ph := range placeholders {
		switch {
		// Case 1: "{}" or "{:x}" without explicit positional index or field
		case ph.PositionalIndex == nil && len(ph.FieldChain) == 0:
			val := getArgOrNoValue(autoIndex, args)
			placeholderValues[i] = val
			autoIndex++

		// Case 2: "{2}", "{1}", etc. (positional, no fields)
		case ph.PositionalIndex != nil && len(ph.FieldChain) == 0:
			val := getArgOrNoValue(*ph.PositionalIndex, args)
			placeholderValues[i] = val

		// Case 3: "{2.Name}", etc. (positional with fields)
		case ph.PositionalIndex != nil && len(ph.FieldChain) > 0:
			baseVal := getArgOrNoValue(*ph.PositionalIndex, args)
			placeholderValues[i] = getFieldChainValue(baseVal, ph.FieldChain)

		// Case 4: No index, but fields => default to argument #0
		case ph.PositionalIndex == nil && len(ph.FieldChain) > 0:
			baseVal := getArgOrNoValue(0, args)
			placeholderValues[i] = getFieldChainValue(baseVal, ph.FieldChain)
		}
	}

	// Build final output
	var sb strings.Builder
	for i, ph := range placeholders {
		sb.WriteString(segments[i]) // literal text
		val := placeholderValues[i]
		formatSpec := placeholderSpecToPrintf(ph.Spec)
		sb.WriteString(fmt.Sprintf(formatSpec, val))
	}
	if len(segments) > len(placeholders) {
		sb.WriteString(segments[len(placeholders)])
	}

	return sb.String()
}

// Printf calls fmt.Print(...) on Sprintf(format, args...).
func Printf(format string, args ...interface{}) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

// Println calls fmt.Println(...) on Sprintf(format, args...).
func Println(format string, args ...interface{}) (int, error) {
	return fmt.Println(Sprintf(format, args...))
}

// Fprintf is like Printf but allows you to specify an io.Writer.
func Fprintf(w io.Writer, format string, args ...interface{}) (int, error) {
	str := Sprintf(format, args...)
	return fmt.Fprint(w, str)
}

// Fprintln is like Println but allows you to specify an io.Writer.
func Fprintln(w io.Writer, format string, args ...interface{}) (int, error) {
	str := Sprintf(format, args...)
	return fmt.Fprintln(w, str)
}

// F quickly formats the string.
func F(format string, args ...interface{}) string {
	return Sprintf(format, args...)
}

// P is a shorthand alternative to Printf
func P(format string, args ...interface{}) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

// Pln is a shorthand alternative to Println
func Pln(format string, args ...interface{}) (int, error) {
	return fmt.Println(Sprintf(format, args...))
}

// ------------------------------------------------------------------
// Parser
// ------------------------------------------------------------------

type placeholder struct {
	PositionalIndex *int
	FieldChain      []string
	Spec            string
}

func parseFormat(format string) ([]string, []placeholder) {
	var segments []string
	var placeholders []placeholder

	r := []rune(format)
	n := len(r)
	var sb strings.Builder

	i := 0
	for i < n {
		switch r[i] {
		case '{':
			// Check escaped '{{'
			if i+1 < n && r[i+1] == '{' {
				sb.WriteRune('{')
				i += 2
				continue
			}
			// Start placeholder
			segments = append(segments, sb.String())
			sb.Reset()

			closing := findClosingBrace(r, i+1)
			if closing == -1 {
				sb.WriteRune('{')
				i++
				continue
			}
			inside := string(r[i+1 : closing])
			i = closing + 1

			ph := parsePlaceholder(inside)
			placeholders = append(placeholders, ph)

		case '}':
			// Check escaped '}}'
			if i+1 < n && r[i+1] == '}' {
				sb.WriteRune('}')
				i += 2
				continue
			}
			sb.WriteRune('}')
			i++

		default:
			sb.WriteRune(r[i])
			i++
		}
	}
	segments = append(segments, sb.String())

	return segments, placeholders
}

func parsePlaceholder(inside string) placeholder {
	// If empty => "{}"
	if inside == "" {
		return placeholder{}
	}
	// If starts with ":" => "{:x}", etc.
	if inside[0] == ':' {
		return placeholder{Spec: inside[1:]}
	}

	// Possibly includes a colon => "0.Name:x"
	colonIdx := strings.IndexRune(inside, ':')
	var mainPart, specPart string
	if colonIdx >= 0 {
		mainPart = inside[:colonIdx]
		specPart = inside[colonIdx+1:]
	} else {
		mainPart = inside
		specPart = ""
	}

	argIndex, fieldChain := parseArgIndexAndFieldChain(mainPart)
	return placeholder{
		PositionalIndex: argIndex,
		FieldChain:      fieldChain,
		Spec:            specPart,
	}
}

func parseArgIndexAndFieldChain(s string) (*int, []string) {
	parts := strings.Split(s, ".")
	if len(parts[0]) > 0 && isAllDigits(parts[0]) {
		idx, err := strconv.Atoi(parts[0])
		if err == nil {
			return &idx, parts[1:]
		}
	}
	return nil, parts
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func findClosingBrace(r []rune, start int) int {
	for j := start; j < len(r); j++ {
		if r[j] == '}' {
			return j
		}
	}
	return -1
}

// ------------------------------------------------------------------
// Field/Map Access
// ------------------------------------------------------------------

func getArgOrNoValue(idx int, args []interface{}) interface{} {
	if idx < 0 || idx >= len(args) {
		return "<no value>"
	}
	return args[idx]
}

func getFieldChainValue(base interface{}, fields []string) interface{} {
	current := base
	for _, f := range fields {
		current = reflectFieldOrMapKey(current, f)
	}
	return current
}

func reflectFieldOrMapKey(val interface{}, name string) interface{} {
	if val == nil {
		return "<invalid field>"
	}
	rv := reflect.ValueOf(val)

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return "<invalid field>"
		}
		rv = rv.Elem()
		fallthrough

	case reflect.Struct:
		return reflectField(rv, name)

	case reflect.Map:
		return reflectMap(rv, name)

	default:
		return "<invalid field>"
	}
}

func reflectField(rv reflect.Value, fieldName string) interface{} {
	fv := rv.FieldByName(fieldName)
	if !fv.IsValid() {
		return "<invalid field>"
	}
	if !fv.CanInterface() {
		return "<invalid field>"
	}
	return fv.Interface()
}

func reflectMap(rv reflect.Value, key string) interface{} {
	if rv.Type().Key().Kind() == reflect.String {
		kv := rv.MapIndex(reflect.ValueOf(key))
		if !kv.IsValid() {
			return "<invalid field>"
		}
		return kv.Interface()
	}
	return "<invalid field>"
}

// ------------------------------------------------------------------
// Format Spec
// ------------------------------------------------------------------

func placeholderSpecToPrintf(spec string) string {
	switch spec {
	case "":
		return "%v"
	case "?":
		return "%+v"
	case "x":
		return "%x"
	case "X":
		return "%X"
	case "b":
		return "%b"
	case "o":
		return "%o"
	case "s":
		return "%s"
	default:
		return "%v"
	}
}
