package fstr

import (
	"errors"
	"fmt"
	"testing"
)

func TestF(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		// Basic formatting
		{
			name:     "simple replacement",
			format:   "Hello, {}!",
			args:     []interface{}{"World"},
			expected: "Hello, World!",
		},
		{
			name:     "multiple replacements",
			format:   "{} + {} = {}",
			args:     []interface{}{1, 2, 3},
			expected: "1 + 2 = 3",
		},

		// Positional arguments
		{
			name:     "positional args",
			format:   "{1} {0} {2}",
			args:     []interface{}{"b", "a", "c"},
			expected: "a b c",
		},
		{
			name:     "repeated positional",
			format:   "{0} and {0}",
			args:     []interface{}{"hello"},
			expected: "hello and hello",
		},

		// Named arguments with map
		{
			name:   "named args map",
			format: "{name} is {age} years old",
			args: []interface{}{map[string]interface{}{
				"name": "Alice",
				"age":  30,
			}},
			expected: "Alice is 30 years old",
		},

		// Named arguments with struct
		{
			name:   "named args struct",
			format: "{Name} is {Age} years old",
			args: []interface{}{struct {
				Name string
				Age  int
			}{"Bob", 25}},
			expected: "Bob is 25 years old",
		},

		// Width and alignment
		{
			name:     "right align",
			format:   "{:>10}",
			args:     []interface{}{"test"},
			expected: "      test",
		},
		{
			name:     "left align",
			format:   "{:<10}",
			args:     []interface{}{"test"},
			expected: "test      ",
		},
		{
			name:     "center align",
			format:   "{:^10}",
			args:     []interface{}{"test"},
			expected: "   test   ",
		},

		// Fill character
		{
			name:     "custom fill",
			format:   "{:0>5}",
			args:     []interface{}{42},
			expected: "00042",
		},

		// Number formatting
		{
			name:     "binary",
			format:   "{:b}",
			args:     []interface{}{42},
			expected: "101010",
		},
		{
			name:     "hex lowercase",
			format:   "{:x}",
			args:     []interface{}{255},
			expected: "ff",
		},
		{
			name:     "hex uppercase",
			format:   "{:X}",
			args:     []interface{}{255},
			expected: "FF",
		},

		// Precision
		{
			name:     "float precision",
			format:   "{:.2}",
			args:     []interface{}{3.14159},
			expected: "3.14",
		},

		// Colors
		{
			name:     "basic color",
			format:   "{|red}{}",
			args:     []interface{}{"error"},
			expected: "\033[31merror\033[0m",
		},
		{
			name:     "bright color",
			format:   "{|bright_blue}{}",
			args:     []interface{}{"info"},
			expected: "\033[94minfo\033[0m",
		},

		// Conditional formatting
		{
			name:     "empty condition true",
			format:   "{:?empty?(empty):(not empty)}",
			args:     []interface{}{""},
			expected: "(empty)",
		},
		{
			name:     "empty condition false",
			format:   "{:?empty?(empty):(not empty)}",
			args:     []interface{}{"test"},
			expected: "(not empty)",
		},
		{
			name:     "comparison condition",
			format:   "{:?>5?(big):(small)}",
			args:     []interface{}{10},
			expected: "(big)",
		},

		// Edge cases
		{
			name:     "nil handling",
			format:   "Value: {}",
			args:     []interface{}{nil},
			expected: "Value: <nil>",
		},
		{
			name:     "empty format string",
			format:   "",
			args:     []interface{}{},
			expected: "",
		},
		{
			name:     "no placeholders",
			format:   "Hello, World!",
			args:     []interface{}{},
			expected: "Hello, World!",
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

func TestE(t *testing.T) {
	err := E("error: {}", "not found")
	if err == nil || err.Error() != "error: not found" {
		t.Errorf("E() error = %v, want %v", err, "error: not found")
	}
}

func TestMust(t *testing.T) {
	// Test successful case
	result := Must("success", nil)
	if result != "success" {
		t.Errorf("Must() = %v, want %v", result, "success")
	}

	// Test panic case
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must() did not panic with error")
		}
	}()
	Must("", errors.New("test error"))
}

// Example tests for documentation
func ExampleF() {
	result := F("Hello, {}!", "World")
	fmt.Println(result)
	// Output: Hello, World!
}

func ExampleF_named() {
	data := map[string]interface{}{
		"name": "Alice",
		"age":  30,
	}
	result := F("{name} is {age} years old", data)
	fmt.Println(result)
	// Output: Alice is 30 years old
}

func ExampleF_alignment() {
	fmt.Println(F("{:<10}", "left"))
	fmt.Println(F("{:>10}", "right"))
	fmt.Println(F("{:^10}", "center"))
	// Output:
	// left
	//      right
	//   center
}

func ExampleF_conditional() {
	empty := ""
	notEmpty := "hello"
	fmt.Println(F("{:?empty?(nothing):(something)}", empty))
	fmt.Println(F("{:?empty?(nothing):(something)}", notEmpty))
	// Output:
	// (nothing)
	// (something)
}
