package fstr

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FormatFunc func(interface{}, FormatSpecifier) string
type VerbFunc func(interface{}) string

var (
	customFormatters     = make(map[reflect.Type]FormatFunc)
	customVerbs          = make(map[string]VerbFunc)
	customFormattersLock sync.RWMutex
	customVerbsLock      sync.RWMutex
)

type FormatSpecifier struct {
	precision   int
	width       int
	alignment   rune
	zeroPad     bool
	sign        rune
	alternate   bool
	typeFlag    rune
	namedArg    string
	positional  int
	color       string
	conditional string
}

// ANSI color codes
var colorCodes = map[string]string{
	"black":   "\033[30m",
	"red":     "\033[31m",
	"green":   "\033[32m",
	"yellow":  "\033[33m",
	"blue":    "\033[34m",
	"magenta": "\033[35m",
	"cyan":    "\033[36m",
	"white":   "\033[37m",
	"reset":   "\033[0m",
}

// Pln prints the formatted string followed by a newline.
func Pln(format string, args ...interface{}) {
	fmt.Println(Format(format, args...))
}

// P prints the formatted string without a newline.
func P(format string, args ...interface{}) {
	fmt.Print(Format(format, args...))
}

// F formats the string and returns the formatted result without printing.
func F(format string, args ...interface{}) string {
	return Format(format, args...)
}

// Format returns the formatted string without printing
func Format(format string, args ...interface{}) string {
	return formatWithNesting(format, args...)
}

func formatWithNesting(format string, args ...interface{}) string {
	re := regexp.MustCompile(`\{([^{}]*)\}`)
	maxDepth := 10 // Prevent infinite recursion

	// Extract named arguments
	namedArgs := make(map[string]interface{})
	for _, arg := range args {
		if m, ok := arg.(map[string]interface{}); ok {
			for k, v := range m {
				namedArgs[k] = v
			}
		}
	}

	var formatArg func(f string, nestedArgs []interface{}, depth int) string
	formatArg = func(f string, nestedArgs []interface{}, depth int) string {
		if depth > maxDepth {
			return f // Return unformatted string if max depth is reached
		}

		return re.ReplaceAllStringFunc(f, func(match string) string {
			spec := parseFormatSpecifier(strings.Trim(match, "{}"))
			var arg interface{}

			if spec.namedArg != "" {
				arg = namedArgs[spec.namedArg]
			} else if spec.positional >= 0 && spec.positional < len(nestedArgs) {
				arg = nestedArgs[spec.positional]
			} else if len(nestedArgs) > 0 {
				arg, nestedArgs = nestedArgs[0], nestedArgs[1:]
			}

			if arg == nil {
				return "" // Return empty string if no matching argument
			}

			formatted := formatSingleArg(arg, spec)

			// Resolve any nested placeholders within the argument's formatting
			if strings.Contains(formatted, "{") && strings.Contains(formatted, "}") {
				formatted = formatArg(formatted, nestedArgs, depth+1)
			}

			// Apply conditional formatting
			if spec.conditional != "" {
				parts := strings.SplitN(spec.conditional, "?", 2)
				condition := evaluateCondition(parts[0], arg)
				if condition && len(parts) > 1 {
					formatted = formatArg(parts[1], nestedArgs, depth+1)
				} else {
					formatted = ""
				}
			}

			// Apply color
			if spec.color != "" {
				formatted = colorize(formatted, spec.color)
			}

			return formatted
		})
	}

	// Initialize the format with the first depth level
	return formatArg(format, args, 0)
}

func formatSingleArg(arg interface{}, spec FormatSpecifier) string {
	v := reflect.ValueOf(arg)
	t := v.Type()

	customFormattersLock.RLock()
	formatter, ok := customFormatters[t]
	customFormattersLock.RUnlock()

	if ok {
		return formatter(arg, spec)
	}

	customVerbsLock.RLock()
	verbFunc, ok := customVerbs[string(spec.typeFlag)]
	customVerbsLock.RUnlock()

	if ok {
		return verbFunc(arg)
	}

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return formatFloat(v.Float(), spec)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return formatInt(v.Int(), spec)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return formatUint(v.Uint(), spec)
	case reflect.String:
		return formatString(v.String(), spec)
	case reflect.Bool:
		return formatBool(v.Bool(), spec)
	case reflect.Slice, reflect.Array:
		return formatSlice(v, spec)
	case reflect.Map:
		return formatMap(v, spec)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return formatTime(v.Interface().(time.Time), spec)
		}
		return formatStruct(v, spec)
	default:
		return fmt.Sprint(arg) // Default formatting
	}
}

func getArgument(spec FormatSpecifier, args *[]interface{}) interface{} {
	if spec.namedArg != "" {
		for _, arg := range *args {
			if m, ok := arg.(map[string]interface{}); ok {
				if val, ok := m[spec.namedArg]; ok {
					return val
				}
			}
		}
	} else if spec.positional >= 0 && spec.positional < len(*args) {
		return (*args)[spec.positional]
	} else if len(*args) > 0 {
		arg := (*args)[0]
		*args = (*args)[1:]
		return arg
	}
	return nil
}

func formatArg(arg interface{}, spec FormatSpecifier) string {
	v := reflect.ValueOf(arg)
	t := v.Type()

	customFormattersLock.RLock()
	formatter, ok := customFormatters[t]
	customFormattersLock.RUnlock()

	if ok {
		return formatter(arg, spec)
	}

	customVerbsLock.RLock()
	verbFunc, ok := customVerbs[string(spec.typeFlag)]
	customVerbsLock.RUnlock()

	if ok {
		return verbFunc(arg)
	}

	var formatted string

	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		formatted = formatFloat(v.Float(), spec)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		formatted = formatInt(v.Int(), spec)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		formatted = formatUint(v.Uint(), spec)
	case reflect.String:
		formatted = formatString(v.String(), spec)
	case reflect.Bool:
		formatted = formatBool(v.Bool(), spec)
	case reflect.Slice, reflect.Array:
		formatted = formatSlice(v, spec)
	case reflect.Map:
		formatted = formatMap(v, spec)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			formatted = formatTime(v.Interface().(time.Time), spec)
		} else {
			formatted = formatStruct(v, spec)
		}
	default:
		formatted = fmt.Sprint(arg) // Default formatting
	}

	return applyWidthAndAlignment(formatted, spec)
}

func applyWidthAndAlignment(s string, spec FormatSpecifier) string {
	if spec.width <= 0 || len(s) >= spec.width {
		return s
	}

	var paddingChar string
	if spec.zeroPad {
		paddingChar = "0"
	} else {
		paddingChar = " "
	}
	padding := strings.Repeat(paddingChar, spec.width-len(s))

	switch spec.alignment {
	case '>':
		return padding + s
	case '^':
		leftPad := padding[:len(padding)/2]
		rightPad := padding[len(padding)/2:]
		return leftPad + s + rightPad
	case '<', 0:
		return s + padding
	default:
		return s
	}
}

// RegisterFormatter registers a custom formatter for a given type.
func RegisterFormatter(t reflect.Type, formatter FormatFunc) {
	customFormattersLock.Lock()
	defer customFormattersLock.Unlock()
	customFormatters[t] = formatter
}

// RegisterVerb registers a custom verb.
func RegisterVerb(t string, verbFunc VerbFunc) {
	customVerbsLock.Lock()
	defer customVerbsLock.Unlock()
	customVerbs[t] = verbFunc
}

func parseFormatSpecifier(spec string) FormatSpecifier {
	fs := FormatSpecifier{precision: -1, width: -1, alignment: 0, sign: 0, positional: -1}

	// Parse color
	if strings.Contains(spec, "|") {
		parts := strings.SplitN(spec, "|", 2)
		fs.color = strings.TrimSpace(parts[1])
		spec = parts[0]
	}

	// Parse conditional
	if strings.Contains(spec, ":?") {
		parts := strings.SplitN(spec, ":?", 2)
		fs.conditional = strings.TrimSpace(parts[1])
		spec = parts[0]
	}

	// Check for named or positional argument
	if strings.Contains(spec, "=") {
		parts := strings.SplitN(spec, "=", 2)
		fs.namedArg = parts[0]
		spec = parts[1]
	} else if pos, err := strconv.Atoi(spec); err == nil {
		fs.positional = pos
		return fs
	}

	// Parse alignment and fill character
	if len(spec) >= 2 && (spec[1] == '<' || spec[1] == '^' || spec[1] == '>') {
		fs.alignment = rune(spec[1])
		fs.zeroPad = spec[0] == '0'
		spec = spec[2:]
	} else if len(spec) >= 1 && (spec[0] == '<' || spec[0] == '^' || spec[0] == '>') {
		fs.alignment = rune(spec[0])
		spec = spec[1:]
	}

	// Parse sign
	if len(spec) > 0 && (spec[0] == '+' || spec[0] == '-' || spec[0] == ' ') {
		fs.sign = rune(spec[0])
		spec = spec[1:]
	}

	// Parse alternate form
	if strings.HasPrefix(spec, "#") {
		fs.alternate = true
		spec = spec[1:]
	}

	// Parse width and precision
	if parts := strings.SplitN(spec, ".", 2); len(parts) > 1 {
		fs.width, _ = strconv.Atoi(parts[0])
		fs.precision, _ = strconv.Atoi(parts[1])
		spec = ""
	} else if w, err := strconv.Atoi(spec); err == nil {
		fs.width = w
		spec = ""
	}

	// Parse type
	if len(spec) > 0 {
		fs.typeFlag = rune(spec[0])
	}

	return fs
}

func colorize(s, color string) string {
	if code, ok := colorCodes[color]; ok && isColorSupported() {
		return code + s + colorCodes["reset"]
	}
	return s
}

func evaluateCondition(condition string, arg interface{}) bool {
	switch condition {
	case "empty":
		return reflect.ValueOf(arg).Len() == 0
	case "nonempty":
		return reflect.ValueOf(arg).Len() > 0
	case "zero":
		return reflect.ValueOf(arg).Interface() == reflect.Zero(reflect.TypeOf(arg)).Interface()
	case "nonzero":
		return reflect.ValueOf(arg).Interface() != reflect.Zero(reflect.TypeOf(arg)).Interface()
	default:
		return false
	}
}

func formatFloat(v float64, spec FormatSpecifier) string {
	var format byte
	var prec int

	switch spec.typeFlag {
	case 'e', 'E':
		format = byte(spec.typeFlag)
		prec = 6
	case 'f', 'F':
		format = 'f'
		prec = 6
	case '%':
		v *= 100
		format = 'f'
		prec = 6
	default:
		format = 'g'
		prec = -1
	}

	if spec.precision > 0 {
		prec = spec.precision
	}

	result := strconv.FormatFloat(v, format, prec, 64)
	if spec.alternate && !strings.Contains(result, ".") {
		result += "."
	}

	return applySign(result, spec, v < 0)
}

func formatInt(v int64, spec FormatSpecifier) string {
	base := 10
	prefix := ""

	switch spec.typeFlag {
	case 'b':
		base = 2
		if spec.alternate {
			prefix = "0b"
		}
	case 'o':
		base = 8
		if spec.alternate {
			prefix = "0o"
		}
	case 'x':
		base = 16
		if spec.alternate {
			prefix = "0x"
		}
	case 'X':
		base = 16
		if spec.alternate {
			prefix = "0X"
		}
	}

	result := prefix + strconv.FormatInt(v, base)

	if spec.typeFlag == 'X' {
		result = strings.ToUpper(result)
	}

	return applySign(result, spec, v < 0)
}

func formatString(v string, spec FormatSpecifier) string {
	if spec.precision >= 0 && len(v) > spec.precision {
		return v[:spec.precision]
	}
	return v
}

func formatBool(v bool, spec FormatSpecifier) string {
	if spec.typeFlag == 'y' {
		if v {
			return "yes"
		}
		return "no"
	}
	return strconv.FormatBool(v)
}

func formatTime(v time.Time, spec FormatSpecifier) string {
	if spec.precision >= 0 {
		return v.Format("2006-01-02 15:04:05.000000000"[:15+spec.precision])
	}
	return v.Format(time.RFC3339)
}

func formatDuration(v time.Duration, spec FormatSpecifier) string {
	if spec.precision >= 0 {
		return v.Round(time.Duration(math.Pow10(9-spec.precision)) * time.Nanosecond).String()
	}
	return v.String()
}

func formatUint(v uint64, spec FormatSpecifier) string {
	base := 10
	prefix := ""

	switch spec.typeFlag {
	case 'b':
		base = 2
		if spec.alternate {
			prefix = "0b"
		}
	case 'o':
		base = 8
		if spec.alternate {
			prefix = "0o"
		}
	case 'x':
		base = 16
		if spec.alternate {
			prefix = "0x"
		}
	case 'X':
		base = 16
		if spec.alternate {
			prefix = "0X"
		}
	}

	result := prefix + strconv.FormatUint(v, base)

	if spec.typeFlag == 'X' {
		result = strings.ToUpper(result)
	}

	return applySign(result, spec, false)
}

func formatSlice(v reflect.Value, spec FormatSpecifier) string {
	var builder strings.Builder
	builder.WriteString("[")
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(formatArg(v.Index(i).Interface(), spec))
	}
	builder.WriteString("]")
	return builder.String()
}

func applySign(s string, spec FormatSpecifier, negative bool) string {
	if !negative {
		switch spec.sign {
		case '+':
			return "+" + s
		case ' ':
			return " " + s
		}
	}
	return s
}

func formatMap(v reflect.Value, spec FormatSpecifier) string {
	var builder strings.Builder
	builder.WriteString("{")
	for i, key := range v.MapKeys() {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(formatArg(key.Interface(), spec))
		builder.WriteString(": ")
		builder.WriteString(formatArg(v.MapIndex(key).Interface(), spec))
	}
	builder.WriteString("}")
	return builder.String()
}

func formatStruct(v reflect.Value, spec FormatSpecifier) string {
	var builder strings.Builder
	t := v.Type()
	builder.WriteString(t.Name())
	builder.WriteString("{")
	for i := 0; i < v.NumField(); i++ {
		if i > 0 {
			builder.WriteString(", ")
		}
		field := t.Field(i)
		builder.WriteString(field.Name)
		builder.WriteString(": ")
		builder.WriteString(formatArg(v.Field(i).Interface(), spec))
	}
	builder.WriteString("}")
	return builder.String()
}

func formatHexDump(data []byte, spec FormatSpecifier) string {
	var builder strings.Builder
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			builder.WriteString("\n")
		} else if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(fmt.Sprintf("%02x", b))
	}
	return builder.String()
}

// ParseFormat parses a format string and returns a function that can be used to format arguments.
func ParseFormat(format string) func(...interface{}) string {
	re := regexp.MustCompile(`\{([^{}]*)\}`)
	placeholders := re.FindAllStringIndex(format, -1)

	return func(args ...interface{}) string {
		result := format
		for i, placeholder := range placeholders {
			if i >= len(args) {
				break
			}
			spec := parseFormatSpecifier(strings.Trim(format[placeholder[0]+1:placeholder[1]-1], "{}"))
			formatted := formatArg(args[i], spec)
			result = result[:placeholder[0]] + formatted + result[placeholder[1]:]
		}
		return result
	}
}

func isColorSupported() bool {
	// Check if the terminal supports color output
	return os.Getenv("TERM") != "dumb" && os.Getenv("NO_COLOR") == ""
}
