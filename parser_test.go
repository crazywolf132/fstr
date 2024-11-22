package fstr

import (
	"reflect"
	"testing"
)

func TestParsePlaceholder(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected placeholder
	}{
		{
			name:    "simple placeholder",
			content: "",
			expected: placeholder{
				position: -1,
			},
		},
		{
			name:    "named argument",
			content: "name",
			expected: placeholder{
				name:     "name",
				position: -1,
			},
		},
		{
			name:    "positional argument",
			content: "0",
			expected: placeholder{
				position: 0,
			},
		},
		{
			name:    "width specification",
			content: ":10",
			expected: placeholder{
				position: -1,
				specifier: FormatSpecifier{
					Width: 10,
				},
			},
		},
		{
			name:    "alignment",
			content: ":>10",
			expected: placeholder{
				position: -1,
				specifier: FormatSpecifier{
					Width:     10,
					Alignment: ">",
				},
			},
		},
		{
			name:    "fill and align",
			content: ":0>10",
			expected: placeholder{
				position: -1,
				specifier: FormatSpecifier{
					Width:     10,
					Alignment: ">",
					Fill:      '0',
				},
			},
		},
		{
			name:    "color",
			content: "|red",
			expected: placeholder{
				position: -1,
				color:    "red",
			},
		},
		{
			name:    "conditional",
			content: "?empty?(none):(value)",
			expected: placeholder{
				position:  -1,
				condition: "empty",
				trueVal:   "(none)",
				falseVal:  "(value)",
			},
		},
		{
			name:    "complex placeholder",
			content: "name:0>10|red?empty?(none):(value)",
			expected: placeholder{
				name:     "name",
				position: -1,
				specifier: FormatSpecifier{
					Width:     10,
					Alignment: ">",
					Fill:      '0',
				},
				color:     "red",
				condition: "empty",
				trueVal:   "(none)",
				falseVal:  "(value)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePlaceholder(tt.content)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("\ngot:  %+v\nwant: %+v", got, tt.expected)
			}
		})
	}
}

func TestParseFormatSpecifier(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected FormatSpecifier
	}{
		{
			name:     "empty spec",
			spec:     "",
			expected: FormatSpecifier{},
		},
		{
			name: "width only",
			spec: "10",
			expected: FormatSpecifier{
				Width: 10,
			},
		},
		{
			name: "precision only",
			spec: ".2",
			expected: FormatSpecifier{
				Precision: 2,
			},
		},
		{
			name: "width and precision",
			spec: "10.2",
			expected: FormatSpecifier{
				Width:     10,
				Precision: 2,
			},
		},
		{
			name: "alignment left",
			spec: "<10",
			expected: FormatSpecifier{
				Width:     10,
				Alignment: "<",
			},
		},
		{
			name: "alignment right",
			spec: ">10",
			expected: FormatSpecifier{
				Width:     10,
				Alignment: ">",
			},
		},
		{
			name: "alignment center",
			spec: "^10",
			expected: FormatSpecifier{
				Width:     10,
				Alignment: "^",
			},
		},
		{
			name: "fill and align",
			spec: "0>10",
			expected: FormatSpecifier{
				Width:     10,
				Alignment: ">",
				Fill:      '0',
			},
		},
		{
			name: "type specifier",
			spec: "b",
			expected: FormatSpecifier{
				Type: "b",
			},
		},
		{
			name: "sign specifier",
			spec: "+10",
			expected: FormatSpecifier{
				Width: 10,
				Sign:  "+",
			},
		},
		{
			name: "alternate form",
			spec: "#x",
			expected: FormatSpecifier{
				Type:      "x",
				Alternate: true,
			},
		},
		{
			name: "complex spec",
			spec: "0>+10.2f",
			expected: FormatSpecifier{
				Width:     10,
				Precision: 2,
				Alignment: ">",
				Fill:      '0',
				Type:      "f",
				Sign:      "+",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFormatSpecifier(tt.spec)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("\ngot:  %+v\nwant: %+v", got, tt.expected)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name             string
		format           string
		wantLiterals     []string
		wantPlaceholders int
	}{
		{
			name:             "empty string",
			format:           "",
			wantLiterals:     []string{},
			wantPlaceholders: 0,
		},
		{
			name:             "no placeholders",
			format:           "Hello, World!",
			wantLiterals:     []string{"Hello, World!"},
			wantPlaceholders: 0,
		},
		{
			name:             "single placeholder",
			format:           "Hello, {}!",
			wantLiterals:     []string{"Hello, ", "!"},
			wantPlaceholders: 1,
		},
		{
			name:             "multiple placeholders",
			format:           "{} + {} = {}",
			wantLiterals:     []string{"", " + ", " = ", ""},
			wantPlaceholders: 3,
		},
		{
			name:             "complex format",
			format:           "Name: {name:>10}, Age: {age:.2f}",
			wantLiterals:     []string{"Name: ", ", Age: ", ""},
			wantPlaceholders: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			literals, placeholders := parse(tt.format)
			if !reflect.DeepEqual(literals, tt.wantLiterals) {
				t.Errorf("literals =\ngot:  %q\nwant: %q", literals, tt.wantLiterals)
			}
			if len(placeholders) != tt.wantPlaceholders {
				t.Errorf("placeholder count = %d, want %d", len(placeholders), tt.wantPlaceholders)
			}
		})
	}
}
