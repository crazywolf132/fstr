# fstr: 🦀 Rust-like String Formatting for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/crazywolf132/fstr.svg)](https://pkg.go.dev/github.com/crazywolf132/fstr)
[![Go Report Card](https://goreportcard.com/badge/github.com/crazywolf132/fstr)](https://goreportcard.com/report/github.com/crazywolf132/fstr)

`fstr` brings Rust's elegant string formatting to Go, with powerful extensions. Format strings with ease using `{}` placeholders, named arguments, colors, and conditional formatting.

## Features

- 🎯 Rust-like `{}` placeholders
- 📝 Named arguments with struct and map support
- 🎨 ANSI color formatting
- 📐 Width, alignment, and precision control
- 🔄 Conditional formatting
- 🎭 Custom formatters and verbs
- 🚦 Type-safe formatting
- ⚡ High performance

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/crazywolf132/fstr"
)

func main() {
    // Basic formatting and printing
    fstr.Pln("Hello, {}! The answer is {}.", "World", 42)

    // Named arguments for more clarity
    fstr.Pln("Hello, {name}! The answer is {answer}.", map[string]interface{}{
        "name":   "Alice",
        "answer": 42,
    })

    // Using colors to enhance output
    fstr.Pln("This is {color|green} and {item|blue}.", map[string]interface{}{
        "color": "green text",
        "item":  "blue text",
    })

    // Conditional formatting based on values
    list := []string{}
    fstr.Pln("The list is {status:?empty?(empty):(not empty)}.", map[string]interface{}{
        "status": list,
    })

    // Returning the formatted string without printing
    formatted := fstr.F("The {0} is {1}!", "answer", 42)
    fmt.Println(formatted) // Outputs: "The answer is 42!"
}
```

### Example Output:

```
Hello, World! The answer is 42.
Hello, Alice! The answer is 42.
This is green text and blue text.
The list is (empty).
The answer is 42!
```

## Comprehensive List of Supported Formatting Options

### 1. **Placeholders**

#### **Basic Placeholder**

- Use `{}` to insert the next argument in order.

```go
fstr.Pln("Hello, {}!", "World") // Output: Hello, World!
```

#### **Named Placeholders**

- Use `{name}` to insert a named argument.

```go
fstr.Pln("Hello, {name}!", map[string]interface{}{
    "name": "Alice",
})
// Output: Hello, Alice!
```

#### **Positional Placeholders**

- Use `{0}`, `{1}`, etc., to refer to positional arguments.

```go
fstr.Pln("{0} + {1} = {2}", 1, 2, 3) // Output: 1 + 2 = 3
```

### 2. **Formatting Options**

#### **Width and Alignment**

- **Width**: `{value:width}` sets the minimum width.

- **Alignment**:
    - **Left-aligned**: `{value:<width}`
    - **Right-aligned**: `{value:>width}`
    - **Center-aligned**: `{value:^width}`

```go
fstr.Pln("Left: |{:<10}|", "left")    // Output: Left: |left      |
fstr.Pln("Right: |{:>10}|", "right")  // Output: Right: |     right|
fstr.Pln("Center: |{:^10}|", "center") // Output: Center: |  center  |
```

#### **Precision**

- For floating-point numbers or strings:

```go
fstr.Pln("Pi: {:.2f}", 3.14159)                // Output: Pi: 3.14
fstr.Pln("Text: {:.5}", "Hello, World!")       // Output: Text: Hello
```

#### **Fill Character**

- Specify a fill character between the `{` and alignment character:

```go
fstr.Pln("Padded: |{:_>10}|", "pad") // Output: Padded: |_______pad|
```

#### **Sign**

- Control the sign for numeric types:

    - **Always show sign**: `{value:+}`
    - **Show space for positive numbers**: `{value: }`

```go
fstr.Pln("Number: {:+}", 42) // Output: Number: +42
fstr.Pln("Number: {: }", 42) // Output: Number:  42
```

#### **Zero Padding**

- Pad numbers with zeros:

```go
fstr.Pln("Zero-padded: {:05}", 42) // Output: Zero-padded: 00042
```

#### **Alternate Form**

- Use `{value:#}` to enable alternate form, e.g., for hex numbers:

```go
fstr.Pln("Hex: {:#x}", 255) // Output: Hex: 0xff
```

### 3. **Number Bases and Formats**

- **Binary**: `{value:b}`
- **Octal**: `{value:o}`
- **Hexadecimal**:
    - Lowercase: `{value:x}`
    - Uppercase: `{value:X}`

```go
fstr.Pln("Binary: {0:b}", 10)        // Output: Binary: 1010
fstr.Pln("Octal: {0:o}", 10)         // Output: Octal: 12
fstr.Pln("Hex (lower): {0:x}", 255)  // Output: Hex (lower): ff
fstr.Pln("Hex (upper): {0:X}", 255)  // Output: Hex (upper): FF
```

### 4. **Floating-Point Formats**

- **Scientific Notation**: `{value:e}` or `{value:E}`
- **Fixed Point**: `{value:f}` or `{value:F}`
- **General Format**: `{value:g}` or `{value:G}`
- **Percentage**: `{value:%}`

```go
fstr.Pln("Scientific: {:.2e}", 12345.6789) // Output: Scientific: 1.23e+04
fstr.Pln("Fixed: {:.2f}", 12345.6789)      // Output: Fixed: 12345.68
fstr.Pln("Percentage: {:.2%}", 0.1234)     // Output: Percentage: 12.34%
```

### 5. **String Case Transformations**

- **Uppercase**: `{value:upper}`
- **Lowercase**: `{value:lower}`

```go
fstr.Pln("Uppercase: {value:upper}", map[string]interface{}{
    "value": "hello",
}) // Output: Uppercase: HELLO
```

### 6. **Conditional Formatting**

- **Syntax**: `{variable:?condition?(true_value):(false_value)}`

- **Conditions**:
    - `empty` - Checks if a string, slice, or map is empty.
    - `nonempty` - Checks if a string, slice, or map is not empty.
    - `zero` - Checks if a numeric value is zero.
    - `nonzero` - Checks if a numeric value is not zero.
    - **Custom conditions** can be implemented as needed.

#### Examples:

```go
list := []int{}
fstr.Pln("List is {list:?empty?(empty):(not empty)}.", map[string]interface{}{
    "list": list,
})
// Output: List is empty.

number := 0
fstr.Pln("Number is {number:?zero?(zero):(non-zero)}.", map[string]interface{}{
    "number": number,
})
// Output: Number is zero.
```

### 7. **Color Formatting**

- **Syntax**: `{value|color}`

- **Supported Colors**: `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`

#### Example:

```go
fstr.Pln("This is {text|red}.", map[string]interface{}{
    "text": "red text",
})
// Output: This is red text. (in red color)
```

### 8. **Escaping Braces**

- Use `{{` and `}}` to include literal `{` and `}` in your strings.

```go
fstr.Pln("Use braces like this: {{ and }}") // Output: Use braces like this: { and }
```

### 9. **Nested Formatting**

- Placeholders can contain other placeholders.

```go
fstr.Pln("Hello, {user{name}}!", map[string]interface{}{
    "user": map[string]interface{}{
        "name": "Alice",
    },
})
// Output: Hello, Alice!
```

### 10. **Custom Formatters**

- You can register custom formatters for specific types.

#### Example:

```go
type Point struct {
    X, Y int
}

fstr.RegisterFormatter(reflect.TypeOf(Point{}), func(arg interface{}, spec fstr.FormatSpecifier) string {
    p := arg.(Point)
    return fmt.Sprintf("(%d, %d)", p.X, p.Y)
})

p := Point{X: 10, Y: 20}
fstr.Pln("Point: {}", p) // Output: Point: (10, 20)
```

### 11. **Formatting Slices, Arrays, Maps, and Structs**

- **Slices/Arrays**: Automatically formatted as `[elem1, elem2, ...]`
- **Maps**: Formatted as `{key1: value1, key2: value2, ...}`
- **Structs**: Formatted with field names and values.

#### Examples:

```go
slice := []int{1, 2, 3}
fstr.Pln("Slice: {}", slice) // Output: Slice: [1, 2, 3]

m := map[string]int{"one": 1, "two": 2}
fstr.Pln("Map: {}", m) // Output: Map: {one: 1, two: 2}

type Person struct {
    Name string
    Age  int
}

person := Person{Name: "Bob", Age: 30}
fstr.Pln("Person: {}", person) // Output: Person: Person{Name: Bob, Age: 30}
```

### 12. **Date and Time Formatting**

- Format `time.Time` values using standard Go format patterns.

#### Example:

```go
import "time"

now := time.Now()
fstr.Pln("Current time: {:%Y-%m-%d %H:%M:%S}", now)
// Output: Current time: 2024-01-01 12:34:56
```

### 13. **Argument Reuse**

- Reuse arguments multiple times without re-specifying them.

```go
fstr.Pln("{0} scored {1} points. Congratulations, {0}!", "Alice", 100)
// Output: Alice scored 100 points. Congratulations, Alice!
```

### 14. **Mixing Positional and Named Arguments**

```go
fstr.Pln("Name: {name}, ID: {0}", 42, map[string]interface{}{
    "name": "Bob",
})
// Output: Name: Bob, ID: 42
```

## Printing vs. Returning Formatted Strings

`fstr` allows you to print or return formatted strings, giving you flexibility based on your use case.

- **Printing**: Use `fstr.Pln()` to print with a newline, or `fstr.P()` to print without a newline.
- **Returning**: Use `fstr.F()` to format and return the string for later use.

#### Example:

```go
formatted := fstr.F("The {0} is {1}!", "answer", 42)
fmt.Println(formatted) // Outputs: "The answer is 42!"
```

## Advanced Usage

### Custom Verbs

You can register custom verbs to extend formatting capabilities.

#### Example:

```go
fstr.RegisterVerb("q", func(arg interface{}) string {
    return fmt.Sprintf("%q", arg)
})

fstr.Pln("Quoted string: {0:q}", "hello") // Output: Quoted string: "hello"
```

### Thread Safety

All shared resources in `fstr` are protected by mutexes, making it safe for concurrent use in goroutines.

## Contributing to fstr

We welcome contributions! 💡 Whether it's a bug report, a feature request, or a pull request, we're happy to hear from you. Visit the [GitHub repository](https://github.com/crazywolf132/fstr) to get started.

## License

`fstr` is licensed under the MIT License.

---

✨ **fstr**: The best of Rust's `println!` for Go, with extra features to make your output beautiful, smart, and flexible!