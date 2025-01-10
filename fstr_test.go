package fstr_test

import (
	"strings"
	"testing"

	"github.com/crazywolf132/fstr"
)

// Person is a simple test struct for named fields.
type Person struct {
	Name   string
	Email  string
	Age    int
	Detail *Detail
}

type Detail struct {
	City string
	Data map[string]int
}

func TestFstr(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		args         []interface{}
		want         string
		partialMatch bool
	}{
		{
			name:   "No_placeholders",
			format: "Hello world",
			args:   nil,
			want:   "Hello world",
		},
		{
			name:   "Basic_auto_placeholders",
			format: "Hello, {}! You have {} new messages.",
			args:   []interface{}{"Alice", 3},
			want:   "Hello, Alice! You have 3 new messages.",
		},
		{
			name:   "Auto_placeholders_missing_arg",
			format: "First: {}, Second: {}, Third: {}",
			args:   []interface{}{"A", "B"},
			want:   "First: A, Second: B, Third: <no value>",
		},
		{
			name:   "Auto_placeholders_extra_arg",
			format: "Used: {}, Unused arg is ignored.",
			args:   []interface{}{"Hello", "ExtraArg"},
			want:   "Used: Hello, Unused arg is ignored.",
		},
		{
			name:   "Positional_placeholders_simple",
			format: "Name: {0}, Age: {1}",
			args:   []interface{}{"Bob", 40},
			want:   "Name: Bob, Age: 40",
		},
		{
			name:   "Positional_placeholders_reorder",
			format: "Second: {1}, First: {0}",
			args:   []interface{}{"A", "B"},
			want:   "Second: B, First: A",
		},
		{
			name:   "Positional_with_missing_out-of-range",
			format: "Val at {2} is out of range, first is {0}",
			args:   []interface{}{"X"},
			want:   "Val at <no value> is out of range, first is X",
		},
		{
			name:   "Named_field_(single_level)_default_arg#0",
			format: "{Name} <{Email}> is {Age} years old",
			args: []interface{}{
				Person{Name: "Charlie", Email: "charlie@example.com", Age: 25},
			},
			want: "Charlie <charlie@example.com> is 25 years old",
		},
		{
			name:   "Named_field_(positional)_with_nested_struct_and_map",
			format: "{0.Detail.City}, data = {0.Detail.Data.info}",
			args: []interface{}{
				Person{
					Name:  "Dana",
					Email: "dana@example.com",
					Age:   30,
					Detail: &Detail{
						City: "Berlin",
						Data: map[string]int{"info": 99},
					},
				},
			},
			want: "Berlin, data = 99",
		},
		{
			name:   "Invalid_field_chain",
			format: "Invalid: {0.Detail.BadField}",
			args: []interface{}{
				Person{
					Name:   "Eve",
					Email:  "eve@example.com",
					Detail: &Detail{City: "New York"},
				},
			},
			want: "Invalid: <invalid field>",
		},
		{
			name:   "Nil_pointer_in_chain",
			format: "Detail city: {0.Detail.City}",
			args: []interface{}{
				Person{Name: "Felix", Email: "felix@example.com", Detail: nil},
			},
			want: "Detail city: <invalid field>",
		},
		{
			name:   "Map_usage_at_top_level_(default_arg_0)_and_subkey",
			format: "User: {user.name}, Count: {count}",
			args: []interface{}{
				map[string]interface{}{
					"user":  map[string]string{"name": "George"},
					"count": 10,
				},
			},
			want: "User: George, Count: 10",
		},
		{
			name:   "Escaped_braces_no_placeholders",
			format: "Escaped open {{ and close }} result: '{{}}'",
			args:   nil,
			want:   "Escaped open { and close } result: '{}'",
		},

		// Updated test: We'll do partial matching, but we'll look for
		// "Hex: ff, Debug: map[" and "one:1" to confirm it's what we expect in Go.
		{
			name:         "Format_spec:_hex,_debug",
			format:       "Hex: {:x}, Debug: {:?}",
			args:         []interface{}{255, map[string]int{"one": 1}},
			want:         "Hex: ff, Debug: map[", // minimal substring
			partialMatch: true,
		},

		{
			name:   "Format_spec:_b,_o,_s",
			format: "Binary: {:b}, Octal: {:o}, String: {:s}",
			args:   []interface{}{15, 15, "Hello"},
			want:   "Binary: 1111, Octal: 17, String: Hello",
		},
		{
			name:   "Positional_with_field_and_spec",
			format: "User hex age: {0.Age:x}",
			args: []interface{}{
				Person{Name: "Hank", Email: "hank@example.com", Age: 31},
			},
			want: "User hex age: 1f",
		},
		{
			name:         "Empty_placeholder_plus_a_field_placeholder",
			format:       "{} & {Name}",
			args:         []interface{}{Person{Name: "Ivy"}},
			partialMatch: true,
			want:         "Ivy", // We'll do a custom check below
		},
		{
			name:   "Zero_arguments,_placeholder_present",
			format: "Just {} no args",
			args:   []interface{}{},
			want:   "Just <no value> no args",
		},
		{
			name:   "Extra_arguments_but_no_placeholders",
			format: "No placeholders here",
			args:   []interface{}{"unused1", 42, "unused2"},
			want:   "No placeholders here",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fstr.Sprintf(tc.format, tc.args...)

			// Handle the "Format_spec:_hex,_debug" test more robustly:
			if tc.name == "Format_spec:_hex,_debug" {
				// We want something like "Hex: ff, Debug: map[one:1]"
				// We'll check two substrings:
				// 1) "Hex: ff, Debug: map["
				// 2) "one:1"
				if !strings.Contains(got, "Hex: ff, Debug: map[") ||
					!strings.Contains(got, "one:1") {
					t.Errorf("got %q, expected partial 'Hex: ff, Debug: map[' and 'one:1'", got)
				}
				return
			}

			// Handle the "Empty_placeholder_plus_a_field_placeholder" test with a custom check:
			if tc.name == "Empty_placeholder_plus_a_field_placeholder" {
				// The struct is printed as something like "{Ivy  0 <nil>} & Ivy".
				// Check that "Ivy" appears at least twice and that " & Ivy" is present.
				if strings.Count(got, "Ivy") < 2 {
					t.Errorf("got %q, expected 'Ivy' at least twice", got)
				}
				if !strings.Contains(got, " & Ivy") {
					t.Errorf("got %q, expected ' & Ivy' substring", got)
				}
				return
			}

			// Otherwise normal checks:
			if tc.partialMatch {
				// Check if 'want' is contained in 'got'
				if !strings.Contains(got, tc.want) {
					t.Errorf("got %q, want substring %q", got, tc.want)
				}
			} else {
				// Exact match
				if got != tc.want {
					t.Errorf("got %q, want %q", got, tc.want)
				}
			}
		})
	}
}
