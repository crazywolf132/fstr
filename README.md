# fstr: 🦀 Rust-like String Formatting for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/crazywolf132/fstr.svg)](https://pkg.go.dev/github.com/crazywolf132/fstr)
[![Go Report Card](https://goreportcard.com/badge/github.com/crazywolf132/fstr)](https://goreportcard.com/report/github.com/crazywolf132/fstr)

`fstr` brings Rust-like string formatting to Go. Format strings with ease using `{}` placeholders, positional arguments, and field access.

## Features

- 🎯 Rust-like `{}` placeholders
- 📝 Positional arguments (`{0}`, `{1}`, etc.)
- 🔍 Field access from structs and maps
- 🎨 Format specifiers for common types
- 🛡️ Graceful handling of mismatched arguments
- ⚡ High performance

## Installation

```bash
go get github.com/crazywolf132/fstr
```

## Version Compatibility

- Requires Go 1.18 or later
- Latest version: Initial Release
- Status: Beta

## Quick Start

```go
package main

import (
    "github.com/crazywolf132/fstr"
)

func main() {
    // Basic auto-incrementing placeholders
    fstr.Pln("Hello, {}!", "World")                 // Output: Hello, World!
    
    // Positional arguments
    fstr.Pln("{1} comes before {0}", "World", "Hello") // Output: Hello comes before World
    
    // Struct field access
    user := struct {
        Name string
        Age  int
    }{"Alice", 30}
    fstr.Pln("Name: {Name}, Age: {Age}", user)     // Output: Name: Alice, Age: 30
    
    // Nested field access
    data := struct {
        User struct {
            Email string
        }
    }{struct{ Email string }{"alice@example.com"}}
    fstr.Pln("Email: {User.Email}", data)          // Output: Email: alice@example.com
    
    // Map field access
    userMap := map[string]interface{}{
        "Name": "Bob",
        "City": "London",
    }
    fstr.Pln("User {Name} from {City}", userMap)   // Output: User Bob from London
    
    // Format specifiers
    fstr.Pln("Hex: {:x}, Binary: {:b}", 255, 10)   // Output: Hex: ff, Binary: 1010
    
    // Debug formatting
    fstr.Pln("Debug view: {:?}", user)             // Output: Debug view: {Name:Alice Age:30}
    
    // Escaping braces
    fstr.Pln("Use {{}} for literal braces")        // Output: Use {} for literal braces
    
    // Graceful handling of missing/extra arguments
    fstr.Pln("Values: {}, {}, {}", 1, 2)           // Output: Values: 1, 2, <no value>
}
```

## Placeholder Syntax

The following placeholder patterns are supported:

- `{}` - Auto-incremented argument (equivalent to `%v`)
- `{0}`, `{1}` - Explicit positional arguments
- `{Name}` - Field "Name" from argument #0 (struct or map)
- `{0.Name}` - Field "Name" from argument #0
- `{Name.Email}` - Nested field access
- `{:x}` - Format specifier for the current argument

## Format Specifiers

Add a format specifier after `:` in any placeholder:

- `{}` - Default formatting (equivalent to `%v`)
- `{:?}` - Debug formatting (equivalent to `%+v`)
- `{:x}` - Lowercase hexadecimal
- `{:X}` - Uppercase hexadecimal
- `{:b}` - Binary
- `{:o}` - Octal
- `{:s}` - String

Format specifiers can be combined with field access:

```go
fstr.Pln("ID: {UserID:x}", map[string]int{"UserID": 255})  // Output: ID: ff
```

## Field Access

Access struct fields or map keys using dot notation:

```go
// Struct example
type User struct {
    Profile struct {
        Email string
    }
}
user := User{Profile: struct{ Email string }{"user@example.com"}}
fstr.Pln("Email: {Profile.Email}", user)  // Output: Email: user@example.com

// Map example
data := map[string]interface{}{
    "user": map[string]string{
        "email": "user@example.com",
    },
}
fstr.Pln("Email: {user.email}", data)     // Output: Email: user@example.com
```

## Escaping Braces

To include literal braces in your output, double them up:

- `{{` produces a literal `{`
- `}}` produces a literal `}`

```go
fstr.Pln("Literal braces: {{ and }}")  // Output: Literal braces: { and }
```

## Argument Handling

The library gracefully handles mismatched argument counts:

- If there are fewer arguments than placeholders, missing values appear as `<no value>`
- If there are more arguments than placeholders, extra arguments are ignored
- If a field doesn't exist, it appears as `<invalid field>`

```go
// Missing argument
fstr.Pln("Need two: {}, {}", 1)        // Output: Need two: 1, <no value>

// Invalid field
fstr.Pln("{NoSuchField}", struct{}{})  // Output: <invalid field>
```

## Available Functions

- `Sprintf(format string, args ...interface{}) string` - Returns formatted string
- `Printf(format string, args ...interface{}) (int, error)` - Prints formatted string
- `Println(format string, args ...interface{}) (int, error)` - Prints formatted string with newline
- `F(format string, args ...interface{}) string` - Shorthand for Sprintf
- `P(format string, args ...interface{}) (int, error)` - Shorthand for Printf
- `Pln(format string, args ...interface{}) (int, error)` - Shorthand for Println
- `Fprintf(w io.Writer, format string, args ...interface{}) (int, error)` - Prints to io.Writer
- `Fprintln(w io.Writer, format string, args ...interface{}) (int, error)` - Prints to io.Writer with newline

## Benchmarks

To run the benchmarks:

```bash
go test -bench=. -benchmem
```

Results produced on a Hetzner CX22:

```bash
$ go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/crazywolf132/fstr
cpu: Intel Xeon Processor (Skylake, IBRS, no TSX)
BenchmarkSimpleString/StringConcat-2         22627114     50.00 ns/op       0 B/op       0 allocs/op
BenchmarkSimpleString/StringBuilder-2         9120834     141.5 ns/op      24 B/op       2 allocs/op
BenchmarkSimpleString/fmt.Sprintf-2           4376912     310.5 ns/op      32 B/op       2 allocs/op
BenchmarkSimpleString/fstr.Sprintf-2           854262      1344 ns/op     184 B/op      11 allocs/op
BenchmarkPositional/fmt.Sprintf-2             4409974     264.0 ns/op      24 B/op       1 allocs/op
BenchmarkPositional/fstr.Sprintf-2             417968      2479 ns/op     424 B/op      18 allocs/op
BenchmarkStructAccess/fmt.Sprintf-2           2917524     416.0 ns/op      40 B/op       2 allocs/op
BenchmarkStructAccess/fstr.Sprintf-2           343814      3361 ns/op     456 B/op      20 allocs/op
BenchmarkNestedAccess/fmt.Sprintf-2           4149006     296.2 ns/op      40 B/op       2 allocs/op
BenchmarkNestedAccess/fstr.Sprintf-2           592467      2142 ns/op     256 B/op      13 allocs/op
BenchmarkMapAccess/fmt.Sprintf-2              4787679     300.0 ns/op      24 B/op       1 allocs/op
BenchmarkMapAccess/fstr.Sprintf-2              390583      3495 ns/op     488 B/op      21 allocs/op
BenchmarkFormatSpecifiers/fmt.Sprintf-2       4351782     335.8 ns/op      32 B/op       1 allocs/op
BenchmarkFormatSpecifiers/fstr.Sprintf-2       461130      2518 ns/op     432 B/op      16 allocs/op
PASS
ok   github.com/crazywolf132/fstr 20.522s
```
