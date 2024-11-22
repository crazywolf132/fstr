package fstr

import (
	"regexp"
	"strconv"
	"strings"
)

// Regular expressions for parsing different placeholder components
var (
	// Matches the entire placeholder: {name:spec|color?cond(true):(false)}
	placeholderRegex = regexp.MustCompile(`\{([^{}]*)\}`)

	// Matches the name/position part: name or 0 in {name:spec} or {0:spec}
	nameRegex = regexp.MustCompile(`^([a-zA-Z0-9_]+|\d+)`)

	// Matches format specifiers: :spec in {name:spec}
	specRegex = regexp.MustCompile(`^:([^|?]*)`)

	// Matches color specifications: |color in {name|red}
	colorRegex = regexp.MustCompile(`\|([a-zA-Z0-9_]+)`)

	// Matches conditional expressions: ?cond(true):(false)
	condRegex = regexp.MustCompile(`\?([^(:]+)\(([^)]*)\):(.*)`)

	// Matches escaped braces: {{ or }}
	escapedBraceRegex = regexp.MustCompile(`{{|}}`)
)

// placeholder represents a parsed format placeholder with all its components
type placeholder struct {
	raw       string          // The original placeholder text
	name      string          // Named argument or empty for positional
	position  int             // Position for positional arguments (-1 for named)
	specifier FormatSpecifier // Formatting rules
	color     string          // ANSI color name if specified
	condition string          // Condition expression
	trueVal   string          // Value to use if condition is true
	falseVal  string          // Value to use if condition is false
	method    string          // Method name to call
}

// parse splits a format string into literal text segments and placeholders
func parse(format string) ([]string, []placeholder) {
	var literals []string
	var placeholders []placeholder

	// First, handle escaped braces by temporarily replacing them
	escapedFormat := escapedBraceRegex.ReplaceAllStringFunc(format, func(match string) string {
		if match == "{{" {
			return "\x00" // Temporary replacement for {{
		}
		return "\x01" // Temporary replacement for }}
	})

	// Track the last position we processed
	lastPos := 0

	// Find all placeholder matches
	matches := placeholderRegex.FindAllStringSubmatchIndex(escapedFormat, -1)

	for _, match := range matches {
		start, end := match[0], match[1]

		// Add the literal text before this placeholder
		if start > lastPos {
			// Restore escaped braces in literal text
			literal := restoreEscapedBraces(escapedFormat[lastPos:start])
			literals = append(literals, literal)
		}

		// Extract and parse the placeholder content
		content := format[start+1 : end-1] // Use original format to parse placeholder
		ph := parsePlaceholder(content)
		ph.raw = format[start:end]
		placeholders = append(placeholders, ph)

		lastPos = end
	}

	// Add any remaining literal text
	if lastPos < len(escapedFormat) {
		// Restore escaped braces in remaining literal text
		literal := restoreEscapedBraces(escapedFormat[lastPos:])
		literals = append(literals, literal)
	}

	return literals, placeholders
}

// Helper function to restore escaped braces
func restoreEscapedBraces(s string) string {
	s = strings.ReplaceAll(s, "\x00", "{")
	s = strings.ReplaceAll(s, "\x01", "}")
	return s
}

// parsePlaceholder breaks down a placeholder's contents into its components
func parsePlaceholder(content string) placeholder {
	ph := placeholder{
		position: -1, // Default to -1 for named arguments
	}

	// Parse name/position
	if nameMatch := nameRegex.FindString(content); nameMatch != "" {
		content = content[len(nameMatch):] // Remove the matched portion

		// Check if it's a positional argument
		if pos, err := strconv.Atoi(nameMatch); err == nil {
			ph.position = pos
		} else {
			ph.name = nameMatch
		}
	}

	// Parse format specifier
	if specMatch := specRegex.FindStringSubmatch(content); len(specMatch) > 1 {
		content = content[len(specMatch[0]):] // Remove the matched portion
		ph.specifier = parseFormatSpecifier(specMatch[1])
	}

	// Parse color
	if colorMatch := colorRegex.FindStringSubmatch(content); len(colorMatch) > 1 {
		content = content[len(colorMatch[0]):] // Remove the matched portion
		ph.color = colorMatch[1]
	}

	// Parse conditional
	if condMatch := condRegex.FindStringSubmatch(content); len(condMatch) > 1 {
		ph.condition = strings.TrimSpace(condMatch[1])
		ph.trueVal = strings.TrimSpace(condMatch[2])
		ph.falseVal = strings.TrimSpace(condMatch[3])
	}

	// Check for method call
	if strings.Contains(ph.name, "()") {
		ph.method = strings.TrimSuffix(ph.name, "()")
		ph.name = strings.TrimSuffix(ph.name, "()")
	}

	return ph
}

// parseFormatSpecifier converts a format specifier string into a FormatSpecifier struct
func parseFormatSpecifier(spec string) FormatSpecifier {
	fs := FormatSpecifier{}

	// Handle empty spec
	if spec == "" {
		return fs
	}

	// Parse alignment and fill character
	if len(spec) >= 2 && (spec[1] == '<' || spec[1] == '>' || spec[1] == '^') {
		fs.Fill = rune(spec[0])
		fs.Alignment = string(spec[1])
		spec = spec[2:]
	} else if len(spec) >= 1 && (spec[0] == '<' || spec[0] == '>' || spec[0] == '^') {
		fs.Alignment = string(spec[0])
		spec = spec[1:]
	}

	// Parse sign
	if len(spec) > 0 && (spec[0] == '+' || spec[0] == '-' || spec[0] == ' ') {
		fs.Sign = string(spec[0])
		spec = spec[1:]
	}

	// Parse alternate form
	if strings.HasPrefix(spec, "#") {
		fs.Alternate = true
		spec = spec[1:]
	}

	// Parse zero padding
	if strings.HasPrefix(spec, "0") {
		fs.ZeroPad = true
		spec = spec[1:]
	}

	// Parse width and precision
	parts := strings.SplitN(spec, ".", 2)
	if width, err := strconv.Atoi(parts[0]); err == nil {
		fs.Width = width
	}

	if len(parts) > 1 {
		if prec, err := strconv.Atoi(parts[1]); err == nil {
			fs.Precision = prec
		}
	}

	// Parse type
	if len(spec) > 0 {
		fs.Type = spec[len(spec)-1:]
	}

	return fs
}
