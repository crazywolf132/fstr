package fstr

import (
	"math"
	"strings"
	"testing"
)

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		// Escaping braces
		{
			name:     "escaped braces",
			format:   "{{}}",
			args:     nil,
			expected: "{}",
		},
		{
			name:     "mixed escaped and placeholder",
			format:   "{{}} {} {{}}",
			args:     []interface{}{"test"},
			expected: "{} test {}",
		},

		// Invalid format strings
		{
			name:     "unclosed brace",
			format:   "Hello {",
			args:     nil,
			expected: "Hello {",
		},
		{
			name:     "unopened brace",
			format:   "Hello }",
			args:     nil,
			expected: "Hello }",
		},

		// Missing arguments
		{
			name:     "missing positional arg",
			format:   "{0} {1} {2}",
			args:     []interface{}{"a", "b"},
			expected: "a b <nil>",
		},
		{
			name:     "missing named arg",
			format:   "{name} {age}",
			args:     []interface{}{map[string]interface{}{"name": "Alice"}},
			expected: "Alice <nil>",
		},

		// Invalid format specifiers
		{
			name:     "invalid alignment",
			format:   "{:@10}",
			args:     []interface{}{"test"},
			expected: "test",
		},
		{
			name:     "invalid precision",
			format:   "{:.abc}",
			args:     []interface{}{3.14},
			expected: "3.14",
		},

		// Nested placeholders
		{
			name:     "nested map access",
			format:   "{user.name}",
			args:     []interface{}{map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}}},
			expected: "Alice",
		},

		// Type mismatches
		{
			name:     "number as string format",
			format:   "{:10s}",
			args:     []interface{}{42},
			expected: "        42",
		},
		{
			name:     "string as number format",
			format:   "{:d}",
			args:     []interface{}{"123"},
			expected: "123",
		},

		// Unicode handling
		{
			name:     "unicode alignment",
			format:   "{:>10}",
			args:     []interface{}{"🦀"},
			expected: "         🦀",
		},
		{
			name:     "unicode fill",
			format:   "{:🦀>5}",
			args:     []interface{}{"Go"},
			expected: "🦀🦀🦀Go",
		},

		// Zero values
		{
			name:     "zero int",
			format:   "{:+05}",
			args:     []interface{}{0},
			expected: "+0000",
		},
		{
			name:     "zero float",
			format:   "{:.2f}",
			args:     []interface{}{0.0},
			expected: "0.00",
		},

		// Special values
		{
			name:     "infinity",
			format:   "{}",
			args:     []interface{}{math.Inf(1)},
			expected: "+Inf",
		},
		{
			name:     "negative infinity",
			format:   "{}",
			args:     []interface{}{math.Inf(-1)},
			expected: "-Inf",
		},
		{
			name:     "NaN",
			format:   "{}",
			args:     []interface{}{math.NaN()},
			expected: "NaN",
		},

		// Complex types
		{
			name:     "channel",
			format:   "{}",
			args:     []interface{}{make(chan int)},
			expected: "<chan>",
		},
		{
			name:     "function",
			format:   "{}",
			args:     []interface{}{func() {}},
			expected: "<func>",
		},

		// Add these test cases to TestEdgeCases
		{
			name:     "double braces as escape",
			format:   "{{}} is literal, {} is placeholder",
			args:     []interface{}{"this"},
			expected: "{} is literal, this is placeholder",
		},
		{
			name:     "multiple escaped braces",
			format:   "{{}} {{}} {{}}",
			args:     nil,
			expected: "{} {} {}",
		},
		{
			name:     "mixed escaped and unescaped",
			format:   "{{name}} = {name}",
			args:     []interface{}{map[string]interface{}{"name": "value"}},
			expected: "{name} = value",
		},
		{
			name:     "json with braces",
			format:   "Data: {{{}}}",
			args:     nil,
			expected: "Data: {}",
		},
		{
			name:     "escaped braces with format specifiers",
			format:   "{{:10}} vs {:10}",
			args:     []interface{}{"test"},
			expected: "{:10} vs      test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := F(tt.format, tt.args...)
			if result != tt.expected {
				t.Errorf("\nformat: %q\n  got:  %q\n  want: %q", tt.format, result, tt.expected)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	// Test concurrent formatting
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			result := F("Test {} {} {}", n, n*2, n*3)
			expected := F("Test {} {} {}", n, n*2, n*3)
			if result != expected {
				t.Errorf("Concurrent formatting failed: got %q, want %q", result, expected)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestRecursion(t *testing.T) {
	// Test recursive formatting
	type Node struct {
		Value    string
		Children []*Node
	}

	root := &Node{
		Value: "root",
		Children: []*Node{
			{Value: "child1"},
			{Value: "child2", Children: []*Node{{Value: "grandchild"}}},
		},
	}

	result := F("{}", root)
	if !strings.Contains(result, "root") || !strings.Contains(result, "child1") ||
		!strings.Contains(result, "child2") || !strings.Contains(result, "grandchild") {
		t.Errorf("Recursive formatting failed: %q", result)
	}
}

func TestCustomError(t *testing.T) {
	type CustomError struct {
		Code    int
		Message string
	}

	err := CustomError{Code: 404, Message: "Not Found"}
	result := F("Error {code}: {message}", err)
	expected := "Error 404: Not Found"
	if result != expected {
		t.Errorf("Custom error formatting failed: got %q, want %q", result, expected)
	}
}
