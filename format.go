package fstr

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// formatArg formats a single argument according to its format specifier
func formatArg(arg interface{}, ph placeholder) string {
	if arg == nil {
		return "<nil>"
	}

	// Check for custom formatter
	if f, ok := GetFormatter(reflect.TypeOf(arg)); ok {
		return f.Format(arg, ph.specifier)
	}

	// Default formatting based on type
	switch v := arg.(type) {
	case string:
		return formatString(v, ph.specifier)
	case int, int8, int16, int32, int64:
		return formatInteger(v, ph.specifier)
	case float32, float64:
		return formatFloat(v, ph.specifier)
	case bool:
		return formatBool(v, ph.specifier)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatString handles string formatting with alignment, padding, and width/precision
func formatString(s string, spec FormatSpecifier) string {
	// Handle precision (truncation)
	if spec.Precision > 0 && len(s) > spec.Precision {
		s = s[:spec.Precision]
	}

	// Handle width and alignment
	width := spec.Width
	if width <= 0 {
		return s
	}

	padding := width - len(s)
	if padding <= 0 {
		return s
	}

	fillChar := spec.Fill
	if fillChar == 0 {
		fillChar = ' '
	}
	pad := strings.Repeat(string(fillChar), padding)

	switch spec.Alignment {
	case "<":
		return s + pad
	case "^":
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(string(fillChar), leftPad) + s + strings.Repeat(string(fillChar), rightPad)
	case ">", "": // right alignment is default
		return pad + s
	default:
		return s // invalid alignment, return unchanged
	}
}

// formatInteger handles integer formatting with base, prefix, padding, and alignment
func formatInteger(i interface{}, spec FormatSpecifier) string {
	var val int64
	switch v := i.(type) {
	case int:
		val = int64(v)
	case int8:
		val = int64(v)
	case int16:
		val = int64(v)
	case int32:
		val = int64(v)
	case int64:
		val = v
	}

	// Handle different bases and prefixes
	var str string
	switch spec.Type {
	case "b":
		str = strconv.FormatInt(val, 2)
		if spec.Alternate {
			str = "0b" + str
		}
	case "o":
		str = strconv.FormatInt(val, 8)
		if spec.Alternate {
			str = "0o" + str
		}
	case "x":
		str = strconv.FormatInt(val, 16)
		if spec.Alternate {
			str = "0x" + str
		}
	case "X":
		str = strings.ToUpper(strconv.FormatInt(val, 16))
		if spec.Alternate {
			str = "0X" + str
		}
	default:
		str = strconv.FormatInt(val, 10)
	}

	// Handle sign
	if val >= 0 {
		switch spec.Sign {
		case "+":
			str = "+" + str
		case " ":
			str = " " + str
		}
	}

	// Handle zero padding
	if spec.ZeroPad && spec.Width > len(str) {
		padding := strings.Repeat("0", spec.Width-len(str))
		return padding + str
	}

	// Use standard string formatting for alignment
	return formatString(str, spec)
}

// formatFloat handles float formatting with precision and scientific notation
func formatFloat(f interface{}, spec FormatSpecifier) string {
	var val float64
	switch v := f.(type) {
	case float32:
		val = float64(v)
	case float64:
		val = v
	}

	precision := spec.Precision
	if precision <= 0 {
		precision = 6 // default precision
	}

	var str string
	switch spec.Type {
	case "e":
		str = strconv.FormatFloat(val, 'e', precision, 64)
	case "E":
		str = strconv.FormatFloat(val, 'E', precision, 64)
	case "f", "F":
		str = strconv.FormatFloat(val, 'f', precision, 64)
	case "g":
		str = strconv.FormatFloat(val, 'g', precision, 64)
	case "G":
		str = strconv.FormatFloat(val, 'G', precision, 64)
	default:
		str = strconv.FormatFloat(val, 'f', precision, 64)
	}

	// Handle sign
	if val >= 0 {
		switch spec.Sign {
		case "+":
			str = "+" + str
		case " ":
			str = " " + str
		}
	}

	return formatString(str, spec)
}

// formatBool handles boolean formatting with custom true/false strings
func formatBool(b bool, spec FormatSpecifier) string {
	var str string
	switch spec.Type {
	case "y", "Y": // yes/no format
		if b {
			str = "yes"
			if spec.Type == "Y" {
				str = "YES"
			}
		} else {
			str = "no"
			if spec.Type == "Y" {
				str = "NO"
			}
		}
	case "t", "T": // true/false format
		if b {
			str = "true"
			if spec.Type == "T" {
				str = "TRUE"
			}
		} else {
			str = "false"
			if spec.Type == "T" {
				str = "FALSE"
			}
		}
	default:
		str = strconv.FormatBool(b)
	}

	return formatString(str, spec)
}

// applyColor applies ANSI color codes to the string
func applyColor(s, color string) string {
	codes := map[string]string{
		"black":   "\033[30m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"reset":   "\033[0m",
		// Bright variants
		"bright_black":   "\033[90m",
		"bright_red":     "\033[91m",
		"bright_green":   "\033[92m",
		"bright_yellow":  "\033[93m",
		"bright_blue":    "\033[94m",
		"bright_magenta": "\033[95m",
		"bright_cyan":    "\033[96m",
		"bright_white":   "\033[97m",
	}

	if code, ok := codes[strings.ToLower(color)]; ok {
		return code + s + codes["reset"]
	}
	return s
}

// applyCondition evaluates a condition and returns the appropriate value
func applyCondition(s, cond, trueVal, falseVal string, arg interface{}) string {
	result := false

	switch cond {
	case "empty":
		result = isEmpty(arg)
	case "!empty":
		result = !isEmpty(arg)
	case "true":
		result = isTruthy(arg)
	case "false":
		result = !isTruthy(arg)
	default:
		// Handle comparison conditions (e.g., ">5", "==foo")
		result = evaluateComparison(cond, arg)
	}

	if result {
		return trueVal
	}
	return falseVal
}

// Helper functions for condition evaluation
func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(val).Int() == 0
	case float32, float64:
		return reflect.ValueOf(val).Float() == 0
	default:
		// Check if it's a slice, array, or map using reflection
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() == 0
		case reflect.Ptr:
			return rv.IsNil()
		default:
			return false
		}
	}
}

func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}

	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(val).Int() != 0
	case float32, float64:
		return reflect.ValueOf(val).Float() != 0
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	default:
		// Check if it's a slice, array, or map using reflection
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		case reflect.Ptr:
			return !rv.IsNil()
		default:
			return true
		}
	}
}

func evaluateComparison(cond string, arg interface{}) bool {
	// Parse comparison operators (==, !=, >, <, >=, <=)
	ops := []string{"==", "!=", ">=", "<=", ">", "<"}
	var op string
	var value string

	for _, o := range ops {
		if strings.Contains(cond, o) {
			parts := strings.SplitN(cond, o, 2)
			if len(parts) == 2 {
				op = o
				value = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if op == "" || value == "" {
		return false
	}

	// Convert arg and value to comparable types
	return compareValues(arg, value, op)
}

func compareValues(arg interface{}, value string, op string) bool {
	switch v := arg.(type) {
	case string:
		return compareStrings(v, value, op)
	case int, int8, int16, int32, int64:
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return compareNumbers(float64(reflect.ValueOf(v).Int()), num, op)
		}
	case float32, float64:
		if num, err := strconv.ParseFloat(value, 64); err == nil {
			return compareNumbers(reflect.ValueOf(v).Float(), num, op)
		}
	case bool:
		if b, err := strconv.ParseBool(value); err == nil {
			return compareBools(v, b, op)
		}
	}
	return false
}

func compareStrings(a, b string, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	default:
		return false
	}
}

func compareNumbers(a, b float64, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	default:
		return false
	}
}

func compareBools(a, b bool, op string) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	default:
		return false
	}
}
