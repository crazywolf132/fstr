# fstr: 🦀 Rust-like `println!` for Go with Superpowers 🚀

**fstr** brings the simplicity and power of Rust's `println!` to Go, and takes it to the next level! Whether you're building quick scripts or complex applications, `fstr` makes string formatting in Go more elegant and expressive. With features like named arguments, custom formatting, and even ANSI color support, you'll wonder how you ever formatted strings without it! 🌈

## Key Features

- **🚀 Rust-style formatting:** `{}` placeholders for easy, readable formatting.
- **🔢 Named and positional arguments:** Format strings with arguments by position or name.
- **🎨 ANSI color support:** Inject colors directly into your formatted output.
- **🌀 Nested and recursive formatting:** Handle dynamic and multi-level formatting with ease.
- **🧠 Conditional formatting:** Apply logic directly in the format string (`empty`, `nonzero`, etc.).
- **🛠 Custom format specifiers:** Extend `fstr` with your own formatting rules for custom types.
- **🗂 Support for complex Go types:** Effortlessly format slices, maps, structs, and more.
- **📦 Flexibility to print or return strings:** Choose whether to print directly or return the formatted string.

## Installation

To add `fstr` to your project, simply run:

```bash
go get github.com/crazywolf132/fstr
```

Or, add it to your `go.mod` file:

```bash
require github.com/crazywolf132/fstr v0.0.1
```

## Quick Start Guide

Start using `fstr` with just a few lines of code! Here's how you can format and print strings with ease:

```go
package main

import (
    "github.com/crazywolf132/fstr"
    "fmt"
)

func main() {
    // Basic formatting and printing
    fstr.Pln("Hello, {0}! The answer is {1}.", "World", 42)

    // Named arguments for more clarity
    fstr.Pln("Hello, {name}! The answer is {answer}.", map[string]interface{}{
        "name":   "Alice",
        "answer": 42,
    })

    // Using colors to enhance output
    fstr.Pln("This is {0|green} and {1|blue}.", "green text", "blue text")

    // Conditional formatting based on values
    list := []string{}
    fstr.Pln("The list is {0:?empty?empty}.", list)

    // Returning the formatted string without printing
    formatted := fstr.F("The {0} is {1}!", "answer", 42)
    fmt.Println(formatted)  // Outputs: "The answer is 42!"
}
```

### Example Output:

```
Hello, World! The answer is 42.
Hello, Alice! The answer is 42.
This is green text and blue text!
The list is empty.
The answer is 42!
```

## Comprehensive List of Supported Formatting Options

### 1. **Positional and Named Arguments**
- **Positional arguments:** `{0}`, `{1}`, etc.
- **Named arguments:** `{name}`, `{age}`, etc.

You can mix and match these argument styles for clarity and flexibility.

#### Examples:

```go
// Positional
fstr.Pln("{0}, you are {1} years old.", "John", 30)

// Named
fstr.Pln("Hello, {name}! You are {age} years old.", map[string]interface{}{
    "name": "John",
    "age":  30,
})
```

### 2. **Custom Format Specifiers**
You can register custom formatters to handle specific types, allowing `fstr` to format them according to your needs.

#### Example:

```go
fstr.RegisterFormatter(reflect.TypeOf(MyType{}), func(arg interface{}, spec fstr.FormatSpecifier) string {
    return fmt.Sprintf("MyType: %v", arg)
})
```

### 3. **Color Formatting**
Inject ANSI colors into your formatted strings. Supported colors:
- `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`

#### Example:

```go
fstr.Pln("This is {0|red} and {1|blue}.", "red text", "blue text")
```

### 4. **Alignment and Padding**
You can specify alignment and padding within placeholders:
- **Left-aligned:** `"<"` (e.g., `"{:<10}"`)
- **Right-aligned:** `">"` (e.g., `"{:>10}"`)
- **Center-aligned:** `"^"` (e.g., `"{:^10}"`)
- **Zero-padded:** Use `0` for padding (e.g., `"{:05}"` for zero-padded numbers)

#### Example:

```go
fstr.Pln("Left: {0:<10}, Right: {1:>10}, Center: {2:^10}", "L", "R", "C")
fstr.Pln("Zero-padded number: {0:05}", 42)
```

### 5. **Width and Precision**
Specify width and precision for formatted output:
- **Width:** Sets the minimum number of characters to display (e.g., `"{:10}"`).
- **Precision:** Limits the number of decimal places or string characters (e.g., `"{:.2}"` for floating points or strings).

#### Example:

```go
fstr.Pln("Number with precision: {0:.2f}", 3.14159)
fstr.Pln("String truncation: {0:.3}", "fstr formatting")
```

### 6. **Conditional Formatting**
Conditionally format your output based on specific conditions:
- **`empty`:** Checks if a slice, map, or string is empty.
- **`nonzero`:** Checks if a number is not zero.
- **`zero`:** Checks if a number is zero.

#### Example:

```go
list := []string{}
fstr.Pln("The list is {0:?empty?empty}.", list)
fstr.Pln("The number is {0:?zero?zero}", 0)
```

### 7. **Nested Formatting**
Format strings that contain placeholders, which themselves contain placeholders.

#### Example:

```go
fstr.Pln("{0}, your balance is {1|green} as of {2}.", "John", 99.99, "2024-01-01")
```

### 8. **Complex Types**
`fstr` automatically handles complex Go types, including:
- **Slices and arrays**: `[1, 2, 3]`
- **Maps**: `{key: value, key2: value2}`
- **Structs**: `{Field1: value1, Field2: value2}`

#### Example:

```go
data := map[string]interface{}{
    "name": "Alice",
    "age":  30,
}
fstr.Pln("{0}", data)  // Output: {name: Alice, age: 30}
```

## Printing vs. Returning Formatted Strings
`fstr` allows you to print or return formatted strings, giving you flexibility based on your use case.

- **Printing**: Use `fstr.Pln()` to print with a newline, or `fstr.P()` to print without a newline.
- **Returning**: Use `fstr.F()` to format and return the string for later use.

#### Example:

```go
formatted := fstr.F("The {0} is {1}!", "answer", 42)
fmt.Println(formatted)  // Outputs: "The answer is 42!"
```

## Contributing to fstr

We welcome contributions! 💡 Whether it's a bug report, a feature request, or a pull request, we're happy to hear from you. Visit the [GitHub repository](https://github.com/crazywolf132/fstr) to get started.

---

✨ **fstr**: The best of Rust's `println!` for Go, with extra features to make your output beautiful, smart, and flexible!
