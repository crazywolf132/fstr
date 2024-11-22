package fstr

import (
	"fmt"
	"testing"
)

func BenchmarkF(b *testing.B) {
	benchmarks := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "simple replacement",
			format: "Hello, {}!",
			args:   []interface{}{"World"},
		},
		{
			name:   "multiple replacements",
			format: "{} + {} = {}",
			args:   []interface{}{1, 2, 3},
		},
		{
			name:   "named arguments",
			format: "{name} is {age} years old",
			args: []interface{}{map[string]interface{}{
				"name": "Alice",
				"age":  30,
			}},
		},
		{
			name:   "complex formatting",
			format: "{:0>10} {|red} {?.empty?(none):(value)}",
			args:   []interface{}{42, "error", ""},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				F(bm.format, bm.args...)
			}
		})
	}
}

func BenchmarkFormatComparison(b *testing.B) {
	// Compare with fmt.Sprintf
	b.Run("fstr.F/simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			F("Hello, {}!", "World")
		}
	})

	b.Run("fmt.Sprintf/simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fmt.Sprintf("Hello, %s!", "World")
		}
	})

	// Compare with complex formatting
	b.Run("fstr.F/complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			F("{name:>10} is {age:.2f} years old", map[string]interface{}{
				"name": "Alice",
				"age":  30.5,
			})
		}
	})

	b.Run("fmt.Sprintf/complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			fmt.Sprintf("%10s is %.2f years old", "Alice", 30.5)
		}
	})
}

func BenchmarkParsing(b *testing.B) {
	benchmarks := []struct {
		name   string
		format string
	}{
		{
			name:   "simple",
			format: "Hello, {}!",
		},
		{
			name:   "named args",
			format: "{name} is {age} years old",
		},
		{
			name:   "format specs",
			format: "{:0>10} {:.2f} {:x}",
		},
		{
			name:   "colors",
			format: "{|red}Error: {|bright_blue}{}",
		},
		{
			name:   "conditionals",
			format: "{:?empty?(none):(value)} {?>10?(big):(small)}",
		},
		{
			name:   "complex mix",
			format: "{name:0>10|red} is {age:.2f} years old {status:?active?(active):(inactive)}",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				parse(bm.format)
			}
		})
	}
}

func BenchmarkFormatters(b *testing.B) {
	specs := FormatSpecifier{
		Width:     10,
		Precision: 2,
		Alignment: ">",
		Fill:      '0',
	}

	benchmarks := []struct {
		name      string
		formatter Formatter
		arg       interface{}
	}{
		{
			name:      "string",
			formatter: StringFormatter{},
			arg:       "test",
		},
		{
			name:      "integer",
			formatter: IntegerFormatter{},
			arg:       12345,
		},
		{
			name:      "float",
			formatter: FloatFormatter{},
			arg:       3.14159,
		},
		{
			name:      "bool",
			formatter: BoolFormatter{},
			arg:       true,
		},
		{
			name:      "slice",
			formatter: SliceFormatter{},
			arg:       []int{1, 2, 3, 4, 5},
		},
		{
			name:      "map",
			formatter: MapFormatter{},
			arg:       map[string]int{"a": 1, "b": 2, "c": 3},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.formatter.Format(bm.arg, specs)
			}
		})
	}
}

func BenchmarkStructFormatting(b *testing.B) {
	type User struct {
		Name     string  `fstr:"name"`
		Age      int     `fstr:"age"`
		Balance  float64 `fstr:"balance"`
		IsActive bool    `fstr:"active"`
	}

	user := User{
		Name:     "Alice",
		Age:      30,
		Balance:  1234.56,
		IsActive: true,
	}

	b.Run("struct_to_map", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			FormatStruct(user)
		}
	})

	b.Run("full_format", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			F("{name} ({age}): ${balance:.2f}", user)
		}
	})
}
