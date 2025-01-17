package fstr

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

var (
	// Pool for string builders to reduce allocations
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// Pool for placeholder slices
	placeholderPool = sync.Pool{
		New: func() interface{} {
			return make([]placeholder, 0, 8)
		},
	}

	// Pool for segment slices
	segmentPool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 8)
		},
	}

	// Cache for reflection field lookups
	fieldCache sync.Map // map[reflect.Type]map[string][]int
)

// Sprintf formats according to a format specifier (with Rust-like placeholders).
func Sprintf(format string, args ...interface{}) string {
	if len(format) == 0 {
		return ""
	}

	// Fast path: if the format contains no braces, return as-is.
	if !strings.ContainsAny(format, "{}") {
		return format
	}

	// Acquire resources from pools
	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	defer builderPool.Put(sb)

	segments := segmentPool.Get().([]string)
	segments = segments[:0]
	defer segmentPool.Put(segments)

	placeholders := placeholderPool.Get().([]placeholder)
	placeholders = placeholders[:0]
	defer placeholderPool.Put(placeholders)

	// Parse format
	segments, placeholders = parseFormatFast(format, segments, placeholders)

	// Pre-allocate builder (slightly over to reduce growth)
	estimatedSize := len(format) + len(args)*10
	sb.Grow(estimatedSize)

	// Now stitch together segments + placeholders
	autoIndex := 0
	for i := 0; i < len(placeholders); i++ {
		// Write the literal chunk before this placeholder
		if i < len(segments) {
			sb.WriteString(segments[i])
		}

		ph := placeholders[i]
		var val interface{}

		switch {
		case ph.PositionalIndex == nil && len(ph.FieldChain) == 0:
			// Automatic ({}), uses next argument
			val = getArgOrNoValue(autoIndex, args)
			autoIndex++
		case ph.PositionalIndex != nil:
			// {0}, {1}...
			val = getArgOrNoValue(*ph.PositionalIndex, args)
			if len(ph.FieldChain) > 0 {
				val = getFieldChainValueFast(val, ph.FieldChain)
			}
		default:
			// Named field from args[0]
			if len(args) > 0 {
				val = getFieldChainValueFast(args[0], ph.FieldChain)
			} else {
				val = "<no value>"
			}
		}

		// Format according to spec
		if ph.Spec == "" {
			// Default formatting
			if s, ok := val.(string); ok {
				sb.WriteString(s)
			} else if stringer, ok := val.(fmt.Stringer); ok {
				sb.WriteString(stringer.String())
			} else {
				formatValFast(sb, val)
			}
		} else {
			// Use custom conversion of spec -> fmt verb
			fmt.Fprintf(sb, placeholderSpecToPrintf(ph.Spec), val)
		}
	}

	// Any leftover segment after the last placeholder
	if len(segments) > len(placeholders) {
		sb.WriteString(segments[len(placeholders)])
	}

	return sb.String()
}

// ----------------------------------------------------------------------------
// Public convenience functions
// ----------------------------------------------------------------------------

func Printf(format string, args ...interface{}) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

func Println(format string, args ...interface{}) (int, error) {
	return fmt.Println(Sprintf(format, args...))
}

func Fprintf(w io.Writer, format string, args ...interface{}) (int, error) {
	str := Sprintf(format, args...)
	return fmt.Fprint(w, str)
}

func Fprintln(w io.Writer, format string, args ...interface{}) (int, error) {
	str := Sprintf(format, args...)
	return fmt.Fprintln(w, str)
}

func F(format string, args ...interface{}) string {
	return Sprintf(format, args...)
}

func P(format string, args ...interface{}) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

func Pln(format string, args ...interface{}) (int, error) {
	return fmt.Println(Sprintf(format, args...))
}

// ----------------------------------------------------------------------------
// Internal placeholder definition
// ----------------------------------------------------------------------------

type placeholder struct {
	PositionalIndex *int
	FieldChain      []string
	Spec            string
}

// ----------------------------------------------------------------------------
// Parsing the format string
// ----------------------------------------------------------------------------

func parseFormatFast(format string, segments []string, placeholders []placeholder) ([]string, []placeholder) {
	data := unsafe.Slice(unsafe.StringData(format), len(format))

	var lastSegment strings.Builder
	lastSegment.Grow(len(data))

	i := 0
	for i < len(data) {
		switch b := data[i]; b {
		case '{':
			// Check for escaped `{{`
			if i+1 < len(data) && data[i+1] == '{' {
				// Write literal '{'
				lastSegment.WriteByte('{')
				i += 2
				continue
			}

			// This is a placeholder start
			// Flush current literal segment (even if empty!)
			segments = append(segments, lastSegment.String())
			lastSegment.Reset()

			// Find matching '}'
			end := findClosingBraceFast(data[i+1:])
			if end == -1 {
				// No closing brace => treat as literal '{'
				lastSegment.WriteByte('{')
				i++
				continue
			}
			end += i + 1 // offset

			// Parse what's inside `{...}`
			ph := parsePlaceholderFast(data[i+1 : end])
			placeholders = append(placeholders, ph)

			i = end + 1 // move past '}'

		case '}':
			// Check for escaped `}}`
			if i+1 < len(data) && data[i+1] == '}' {
				// Write literal '}'
				lastSegment.WriteByte('}')
				i += 2
				continue
			}
			// Otherwise literal '}'
			lastSegment.WriteByte('}')
			i++

		default:
			// Normal character
			lastSegment.WriteByte(b)
			i++
		}
	}

	// Flush whatever's left
	if lastSegment.Len() > 0 {
		segments = append(segments, lastSegment.String())
	}
	return segments, placeholders
}

// Return the offset of the first '}' in data, or -1 if not found
func findClosingBraceFast(data []byte) int {
	for i, b := range data {
		if b == '}' {
			return i
		}
	}
	return -1
}

// Parse the inside of `{ ... }` into a placeholder: optional positional index,
// optional field chain, optional spec after ':'
func parsePlaceholderFast(data []byte) placeholder {
	var ph placeholder
	if len(data) == 0 {
		return ph
	}

	// See if we have a ':' for a spec
	specStart := -1
	for i := 0; i < len(data); i++ {
		if data[i] == ':' {
			specStart = i
			break
		}
	}

	var fieldPart []byte
	if specStart >= 0 {
		fieldPart = data[:specStart]
		ph.Spec = string(data[specStart+1:])
	} else {
		fieldPart = data
	}

	// If no field part, just return
	if len(fieldPart) == 0 {
		return ph
	}

	// If it starts with a digit, treat it as a positional index
	if fieldPart[0] >= '0' && fieldPart[0] <= '9' {
		var idx int
		for i := 0; i < len(fieldPart); i++ {
			if fieldPart[i] == '.' {
				// e.g. "0.Detail"
				idx64, err := strconv.ParseInt(string(fieldPart[:i]), 10, 64)
				if err != nil {
					return ph
				}
				idx = int(idx64)
				ph.PositionalIndex = &idx
				ph.FieldChain = strings.Split(string(fieldPart[i+1:]), ".")
				return ph
			}
			if fieldPart[i] < '0' || fieldPart[i] > '9' {
				// invalid => bail out
				return ph
			}
		}
		// purely numeric
		idx64, err := strconv.ParseInt(string(fieldPart), 10, 64)
		if err != nil {
			return ph
		}
		idx = int(idx64)
		ph.PositionalIndex = &idx
		return ph
	}

	// Otherwise it's a named field chain
	ph.FieldChain = strings.Split(string(fieldPart), ".")
	return ph
}

// ----------------------------------------------------------------------------
// Formatting logic
// ----------------------------------------------------------------------------

func formatValFast(sb *strings.Builder, val interface{}) {
	if val == nil {
		sb.WriteString("<no value>")
		return
	}

	switch v := val.(type) {
	case string:
		sb.WriteString(v)
	case int:
		writeInt(sb, int64(v))
	case int8:
		writeInt(sb, int64(v))
	case int16:
		writeInt(sb, int64(v))
	case int32:
		writeInt(sb, int64(v))
	case int64:
		writeInt(sb, v)
	case uint:
		writeUint(sb, uint64(v))
	case uint8:
		writeUint(sb, uint64(v))
	case uint16:
		writeUint(sb, uint64(v))
	case uint32:
		writeUint(sb, uint64(v))
	case uint64:
		writeUint(sb, v)
	case bool:
		if v {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case float32:
		sb.WriteString(strconv.FormatFloat(float64(v), 'f', -1, 32))
	case float64:
		sb.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
	case fmt.Stringer:
		sb.WriteString(v.String())
	case error:
		sb.WriteString(v.Error())
	default:
		// Handle struct or fallback
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			// If no fmt.Stringer is implemented, we check for a "Name" field
			nameField := rv.FieldByName("Name")
			if nameField.IsValid() && nameField.Kind() == reflect.String {
				sb.WriteString(nameField.String())
			} else {
				// fallback to default
				fmt.Fprintf(sb, "%v", val)
			}
		} else {
			// fallback to default
			fmt.Fprintf(sb, "%v", val)
		}
	}
}

// Manual writing of int64 -> decimal ASCII
func writeInt(sb *strings.Builder, i int64) {
	if i == 0 {
		sb.WriteByte('0')
		return
	}
	var buf [20]byte
	var length int
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		buf[length] = byte(i%10 + '0')
		length++
		i /= 10
	}
	if neg {
		sb.WriteByte('-')
	}
	for j := length - 1; j >= 0; j-- {
		sb.WriteByte(buf[j])
	}
}

// Manual writing of uint64 -> decimal ASCII
func writeUint(sb *strings.Builder, u uint64) {
	if u == 0 {
		sb.WriteByte('0')
		return
	}
	var buf [20]byte
	var length int
	for u > 0 {
		buf[length] = byte(u%10 + '0')
		length++
		u /= 10
	}
	for j := length - 1; j >= 0; j-- {
		sb.WriteByte(buf[j])
	}
}

// ----------------------------------------------------------------------------
// Field chain resolution (reflection or map lookup)
// ----------------------------------------------------------------------------

func getFieldChainValueFast(base interface{}, fields []string) interface{} {
	if base == nil || len(fields) == 0 {
		return "<invalid field>"
	}

	val := reflect.ValueOf(base)
	typ := val.Type()

	// Handle pointers
	for typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "<invalid field>"
		}
		val = val.Elem()
		typ = val.Type()
	}

	// If top-level is a map[string]..., handle that
	if typ.Kind() == reflect.Map && typ.Key().Kind() == reflect.String {
		if len(fields) == 1 {
			v := val.MapIndex(reflect.ValueOf(fields[0]))
			if !v.IsValid() {
				return "<invalid field>"
			}
			if v.Kind() == reflect.Interface {
				v = v.Elem()
			}
			if !v.IsValid() || !v.CanInterface() {
				return "<invalid field>"
			}
			return v.Interface()
		}
		// recursively handle nested
		v := val.MapIndex(reflect.ValueOf(fields[0]))
		if !v.IsValid() {
			return "<invalid field>"
		}
		if v.Kind() == reflect.Interface {
			v = v.Elem()
		}
		if !v.IsValid() {
			return "<invalid field>"
		}
		return getFieldChainValueFast(v.Interface(), fields[1:])
	}

	// Otherwise treat as a struct or nested map
	for i, field := range fields {
		switch val.Kind() {
		case reflect.Struct:
			// Attempt direct field by name
			f := val.FieldByName(field)
			if !f.IsValid() {
				// Try cached field indices
				idx, ok := getCachedFieldIndices(typ, field)
				if !ok {
					return "<invalid field>"
				}
				f = val.FieldByIndex(idx)
				if !f.IsValid() {
					return "<invalid field>"
				}
			}
			val = f

		case reflect.Map:
			if val.Type().Key().Kind() != reflect.String {
				return "<invalid field>"
			}
			val = val.MapIndex(reflect.ValueOf(field))
			if !val.IsValid() {
				return "<invalid field>"
			}

		default:
			return "<invalid field>"
		}

		// handle pointer at each step
		for val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return "<invalid field>"
			}
			val = val.Elem()
		}
		typ = val.Type()

		// if this is the last field
		if i == len(fields)-1 {
			if !val.IsValid() {
				return "<invalid field>"
			}
			if val.Kind() == reflect.Interface {
				val = val.Elem()
			}
		}
	}

	if !val.IsValid() || !val.CanInterface() {
		return "<invalid field>"
	}
	return val.Interface()
}

// Cached reflection field indices
func getCachedFieldIndices(typ reflect.Type, fieldName string) ([]int, bool) {
	if cached, ok := fieldCache.Load(typ); ok {
		if indices, ok2 := cached.(map[string][]int)[fieldName]; ok2 {
			return indices, true
		}
	}
	cache := make(map[string][]int)
	buildFieldCache(typ, nil, cache)
	fieldCache.Store(typ, cache)

	indices, ok := cache[fieldName]
	return indices, ok
}

func buildFieldCache(typ reflect.Type, prefix []int, cache map[string][]int) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		idx := append(prefix, i)
		cache[f.Name] = append([]int(nil), idx...)
		// If nested struct, recurse
		if f.Type.Kind() == reflect.Struct {
			buildFieldCache(f.Type, idx, cache)
		} else if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
			buildFieldCache(f.Type.Elem(), idx, cache)
		}
	}
}

// ----------------------------------------------------------------------------
// Argument retrieval
// ----------------------------------------------------------------------------

func getArgOrNoValue(idx int, args []interface{}) interface{} {
	if idx < 0 || idx >= len(args) {
		return "<no value>"
	}
	return args[idx]
}

// ----------------------------------------------------------------------------
// Converting our spec to a fmt.Fprintf format string
// ----------------------------------------------------------------------------

func placeholderSpecToPrintf(spec string) string {
	// Basic mapping from Rust-like spec to Go verbs
	switch spec {
	case "":
		return "%v"
	case "?":
		return "%+v" // debug-like
	case "x", "X", "o", "O", "b", "B", "d", "c", "U":
		// e.g. {:x}, {:b}, {:o}, ...
		return "%" + spec
	case "e", "E", "f", "F", "g", "G":
		return "%" + spec
	case "s":
		return "%s"
	default:
		return "%" + spec
	}
}
