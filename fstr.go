package fstr

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
)

// formatWithMap handles format strings with named arguments
func formatWithMap(format string, args map[string]interface{}) string {
	literals, placeholders := parse(format)
	var result strings.Builder

	for i, literal := range literals {
		result.WriteString(literal)

		if i < len(placeholders) {
			ph := placeholders[i]
			var arg interface{}

			// Get named argument
			if ph.name != "" {
				arg = args[ph.name]
			}

			// Format the argument
			if arg != nil {
				formatted := formatArg(arg, ph)

				// Apply color if specified
				if ph.color != "" {
					formatted = applyColor(formatted, ph.color)
				}

				// Apply conditional formatting if specified
				if ph.condition != "" {
					formatted = applyCondition(formatted, ph.condition, ph.trueVal, ph.falseVal, arg)
				}

				result.WriteString(formatted)
			} else {
				result.WriteString("<nil>")
			}
		}
	}

	return result.String()
}

// F formats a string using the provided format string and arguments.
// It supports named arguments, positional arguments, format specifiers,
// colors, and conditional formatting.
//
// Basic usage:
//
//	fstr.F("Hello, {}!", "World")                // "Hello, World!"
//	fstr.F("The answer is {}", 42)              // "The answer is 42"
//
// Named arguments:
//
//	fstr.F("Hello, {name}!", map[string]any{"name": "Alice"})
//	// or with a struct:
//	fstr.F("Hello, {name}!", struct{name string}{"Alice"})
//
// Format specifiers:
//
//	fstr.F("{:>10}", "right")                   // "     right"
//	fstr.F("{:<10}", "left")                    // "left     "
//	fstr.F("{:^10}", "center")                  // "  center  "
//	fstr.F("{:0>3}", 5)                         // "005"
//	fstr.F("{:.2}", 3.14159)                    // "3.14"
//
// Colors:
//
//	fstr.F("{|red}Error: {}", "not found")      // Red "Error: not found"
//	fstr.F("{|bright_blue}Info: {}", "ready")   // Bright blue "Info: ready"
//
// Conditional formatting:
//
//	fstr.F("{:?empty?(none):(value)}", "")      // "(none)"
//	fstr.F("{:?>10?(big):(small)}", 42)         // "(small)"
func F(format string, args ...interface{}) string {
	// Get cached or parse format string
	pf := getParsedFormat(format)

	// Handle single argument struct/map case
	if len(args) == 1 {
		switch v := args[0].(type) {
		case map[string]interface{}:
			return formatWithParsed(pf, v)
		default:
			if reflect.TypeOf(v).Kind() == reflect.Struct {
				return formatWithParsed(pf, FormatStruct(v))
			}
		}
	}

	var result strings.Builder
	argIndex := 0

	for i, literal := range pf.literals {
		result.WriteString(literal)

		if i < len(pf.placeholders) {
			ph := pf.placeholders[i]
			var arg interface{}

			if ph.position >= 0 && ph.position < len(args) {
				arg = args[ph.position]
			} else if argIndex < len(args) {
				arg = args[argIndex]
				argIndex++
			}

			if arg != nil {
				// Handle nested field access
				if paths, ok := pf.accessPaths[ph.name]; ok {
					arg = getNestedValue(arg, paths)
				}

				formatted := formatArg(arg, ph)
				if ph.color != "" {
					formatted = applyColor(formatted, ph.color)
				}
				if ph.condition != "" {
					formatted = applyCondition(formatted, ph.condition, ph.trueVal, ph.falseVal, arg)
				}
				result.WriteString(formatted)
			} else {
				result.WriteString("<nil>")
			}
		}
	}

	return result.String()
}

// formatWithParsed formats using a pre-parsed format
func formatWithParsed(pf parsedFormat, args map[string]interface{}) string {
	var result strings.Builder

	for i, literal := range pf.literals {
		result.WriteString(literal)

		if i < len(pf.placeholders) {
			ph := pf.placeholders[i]
			var arg interface{}

			// Handle nested field access
			if paths, ok := pf.accessPaths[ph.name]; ok {
				arg = getNestedValue(args, paths)
			} else if ph.name != "" {
				arg = args[ph.name]
			}

			if arg != nil {
				formatted := formatArg(arg, ph)
				if ph.color != "" {
					formatted = applyColor(formatted, ph.color)
				}
				if ph.condition != "" {
					formatted = applyCondition(formatted, ph.condition, ph.trueVal, ph.falseVal, arg)
				}
				result.WriteString(formatted)
			} else {
				result.WriteString("<nil>")
			}
		}
	}

	return result.String()
}

// P prints the formatted string to stdout without a newline.
//
// Example:
//
//	fstr.P("Loading... {}%", 50)
func P(format string, args ...interface{}) {
	fmt.Fprint(os.Stdout, F(format, args...))
}

// Pln prints the formatted string to stdout with a newline.
//
// Example:
//
//	fstr.Pln("User {} logged in", "Alice")
func Pln(format string, args ...interface{}) {
	fmt.Fprintln(os.Stdout, F(format, args...))
}

// E returns the formatted string as an error.
//
// Example:
//
//	return fstr.E("invalid value: {}", val)
func E(format string, args ...interface{}) error {
	return fmt.Errorf(F(format, args...))
}

// Err prints the formatted error string to stderr.
//
// Example:
//
//	fstr.Err("{|red}Error: {}", err)
func Err(format string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, F(format, args...))
}

// Debug prints the formatted string to stderr with file and line information.
//
// Example:
//
//	fstr.Debug("Processing value: {}", x)
//	// Output: main.go:42: Processing value: foo
func Debug(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	if lastSlash := strings.LastIndex(file, "/"); lastSlash >= 0 {
		file = file[lastSlash+1:]
	}

	fmt.Fprintf(os.Stderr, "%s:%d: %s\n", file, line, F(format, args...))
}

// Must is a helper that wraps a call to a function returning (string, error)
// and panics if the error is non-nil.
//
// Example:
//
//	path := fstr.Must(filepath.Abs("./config"))
func Must(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

// FormatSpecifier holds the formatting rules for a placeholder
type FormatSpecifier struct {
	Width     int    // Minimum width of the formatted string
	Precision int    // Precision for floating-point numbers
	Alignment string // '<' (left), '^' (center), '>' (right)
	Fill      rune   // Character to use for padding
	Type      string // Format type (b, x, o, etc.)
	Alternate bool   // Use alternate form (#)
	Sign      string // Sign control (+, -, space)
	ZeroPad   bool   // Use zero padding for numbers
}

// Formatter interface defines how custom types can implement their own formatting
type Formatter interface {
	Format(arg interface{}, spec FormatSpecifier) string
}

// Add this function
func Safe(format string, args ...interface{}) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = "<format error>"
			err = fmt.Errorf("format error: %v", r)
		}
	}()

	result = F(format, args...)
	return result, nil
}

// Add this function
func Env(format string, args ...interface{}) string {
	result := F(format, args...)
	return os.ExpandEnv(result)
}
