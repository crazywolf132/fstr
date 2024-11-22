package fstr

import (
	"reflect"
	"testing"
)

func TestFormatStruct(t *testing.T) {
	type Embedded struct {
		Value string `fstr:"embedded_value"`
	}

	type TestStruct struct {
		Name       string `fstr:"name"`
		Age        int    `fstr:"age"`
		Private    string `fstr:"-"`
		unexported string
		NoTag      string
		Embedded
	}

	tests := []struct {
		name     string
		input    interface{}
		expected map[string]interface{}
	}{
		{
			name: "basic struct",
			input: TestStruct{
				Name:       "Alice",
				Age:        30,
				Private:    "hidden",
				unexported: "not visible",
				NoTag:      "visible",
				Embedded:   Embedded{Value: "embedded"},
			},
			expected: map[string]interface{}{
				"name":           "Alice",
				"age":            30,
				"NoTag":          "visible",
				"embedded_value": "embedded",
			},
		},
		{
			name:     "nil pointer",
			input:    (*TestStruct)(nil),
			expected: map[string]interface{}{},
		},
		{
			name: "pointer to struct",
			input: &TestStruct{
				Name: "Bob",
				Age:  25,
			},
			expected: map[string]interface{}{
				"name":           "Bob",
				"age":            25,
				"NoTag":          "",
				"embedded_value": "",
			},
		},
		{
			name:     "non-struct",
			input:    "not a struct",
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStruct(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("\ngot:  %+v\nwant: %+v", got, tt.expected)
			}
		})
	}
}

func TestDefaultFormatters(t *testing.T) {
	tests := []struct {
		name      string
		formatter Formatter
		arg       interface{}
		spec      FormatSpecifier
		expected  string
	}{
		{
			name:      "string formatter basic",
			formatter: StringFormatter{},
			arg:       "test",
			spec:      FormatSpecifier{},
			expected:  "test",
		},
		{
			name:      "string formatter width",
			formatter: StringFormatter{},
			arg:       "test",
			spec:      FormatSpecifier{Width: 8, Alignment: ">"},
			expected:  "    test",
		},
		{
			name:      "integer formatter basic",
			formatter: IntegerFormatter{},
			arg:       42,
			spec:      FormatSpecifier{},
			expected:  "42",
		},
		{
			name:      "integer formatter hex",
			formatter: IntegerFormatter{},
			arg:       255,
			spec:      FormatSpecifier{Type: "x"},
			expected:  "ff",
		},
		{
			name:      "float formatter basic",
			formatter: FloatFormatter{},
			arg:       3.14159,
			spec:      FormatSpecifier{Precision: 2},
			expected:  "3.14",
		},
		{
			name:      "bool formatter basic",
			formatter: BoolFormatter{},
			arg:       true,
			spec:      FormatSpecifier{},
			expected:  "true",
		},
		{
			name:      "bool formatter yes/no",
			formatter: BoolFormatter{},
			arg:       true,
			spec:      FormatSpecifier{Type: "y"},
			expected:  "yes",
		},
		{
			name:      "slice formatter",
			formatter: SliceFormatter{},
			arg:       []int{1, 2, 3},
			spec:      FormatSpecifier{},
			expected:  "[1, 2, 3]",
		},
		{
			name:      "map formatter",
			formatter: MapFormatter{},
			arg:       map[string]int{"a": 1, "b": 2},
			spec:      FormatSpecifier{},
			expected:  "{a: 1, b: 2}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.formatter.Format(tt.arg, tt.spec)
			if got != tt.expected {
				t.Errorf("\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}

func TestCustomVerbs(t *testing.T) {
	tests := []struct {
		name     string
		verb     string
		arg      interface{}
		spec     FormatSpecifier
		expected string
	}{
		{
			name:     "json basic",
			verb:     "json",
			arg:      map[string]string{"name": "Alice"},
			expected: `{"name":"Alice"}`,
		},
		{
			name:     "hex number",
			verb:     "hex",
			arg:      255,
			expected: "ff",
		},
		{
			name:     "binary number",
			verb:     "bin",
			arg:      42,
			expected: "101010",
		},
		{
			name:     "upper string",
			verb:     "upper",
			arg:      "hello",
			expected: "HELLO",
		},
		{
			name:     "lower string",
			verb:     "lower",
			arg:      "HELLO",
			expected: "hello",
		},
	}

	// Register test verbs
	RegisterVerb("json", jsonVerb, "Format as JSON")
	RegisterVerb("hex", hexVerb, "Format as hexadecimal")
	RegisterVerb("bin", binVerb, "Format as binary")
	RegisterVerb("upper", upperVerb, "Convert to uppercase")
	RegisterVerb("lower", lowerVerb, "Convert to lowercase")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verb, ok := GetFormattingVerb(tt.verb)
			if !ok {
				t.Fatalf("verb %q not found", tt.verb)
			}

			got := verb.handler(tt.arg, tt.spec)
			if got != tt.expected {
				t.Errorf("\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}

func TestFormatterRegistration(t *testing.T) {
	type Custom struct {
		value string
	}

	customFormatter := StringFormatter{}
	customType := reflect.TypeOf(Custom{})

	// Test registration
	RegisterFormatter(customType, customFormatter)

	// Test retrieval
	got, ok := GetFormatter(customType)
	if !ok {
		t.Error("formatter not found after registration")
	}

	if !reflect.DeepEqual(got, customFormatter) {
		t.Errorf("got formatter %v, want %v", got, customFormatter)
	}

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			RegisterFormatter(customType, customFormatter)
			_, _ = GetFormatter(customType)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestVerbRegistration(t *testing.T) {
	testVerb := "test"
	testHandler := func(v interface{}, spec FormatSpecifier) string {
		return "test"
	}
	testDocs := "Test verb"

	// Test registration
	RegisterVerb(testVerb, testHandler, testDocs)

	// Test retrieval
	verb, ok := GetFormattingVerb(testVerb)
	if !ok {
		t.Error("verb not found after registration")
	}

	if verb.docs != testDocs {
		t.Errorf("got docs %q, want %q", verb.docs, testDocs)
	}

	result := verb.handler("test", FormatSpecifier{})
	if result != "test" {
		t.Errorf("got result %q, want %q", result, "test")
	}

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			RegisterVerb(testVerb, testHandler, testDocs)
			_, _ = GetFormattingVerb(testVerb)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
