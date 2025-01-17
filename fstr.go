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

// ----------------------------------------------------------------------
// Global parse cache for repeated format strings
// ----------------------------------------------------------------------

type parsedResult struct {
	segments     []string
	placeholders []placeholder
}

// formatCache caches the parse result (segments + placeholders) for each
// unique format string. Key = format string, Value = *parsedResult.
var formatCache sync.Map

// ----------------------------------------------------------------------
// Pools
// ----------------------------------------------------------------------

var (
	// Pool for string builders to reduce allocations
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// We still keep a pool for placeholders, but note that once we start
	// caching parse results, we won’t need it for subsequent calls. The
	// placeholders will live in the cache. However, for new format strings,
	// we still use it.
	placeholderPool = sync.Pool{
		New: func() interface{} {
			return make([]placeholder, 0, 8)
		},
	}

	// Similarly, a pool for segments
	segmentPool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 8)
		},
	}

	// Reflection field lookups
	fieldCache sync.Map // map[reflect.Type]map[string][]int
)

// ----------------------------------------------------------------------
// Public entry points
// ----------------------------------------------------------------------

// Sprintf formats according to a format specifier (with Rust-like placeholders).
func Sprintf(format string, args ...interface{}) string {
	if len(format) == 0 {
		return ""
	}
	// Fast path: if the format has no braces, we can short-circuit
	if !strings.ContainsAny(format, "{}") {
		return format
	}

	// Check the parse cache
	cacheVal, found := formatCache.Load(format)
	var pr *parsedResult
	if found {
		pr = cacheVal.(*parsedResult)
	} else {
		// Need to parse
		pr = parseFormatAndCache(format)
	}

	// Now do the rendering
	return renderSprintf(pr, args)
}

// Provide short-hand convenience functions
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

// Short aliases
func F(format string, args ...interface{}) string {
	return Sprintf(format, args...)
}

func P(format string, args ...interface{}) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

func Pln(format string, args ...interface{}) (int, error) {
	return fmt.Println(Sprintf(format, args...))
}

// ----------------------------------------------------------------------
// Placeholder structure
// ----------------------------------------------------------------------

type placeholder struct {
	PositionalIndex *int
	FieldChain      []string
	// We store the actual Go "fmt" verb here, e.g. "%v", "%s", "%x", etc.
	GoFmtVerb string
}

// ----------------------------------------------------------------------
// The "cached parse" step
// ----------------------------------------------------------------------

func parseFormatAndCache(format string) *parsedResult {
	segments := segmentPool.Get().([]string)
	segments = segments[:0] // reset
	defer segmentPool.Put(segments)

	phs := placeholderPool.Get().([]placeholder)
	phs = phs[:0]
	defer placeholderPool.Put(phs)

	segments, phs = parseFormatFast(format, segments, phs)

	// Build a parsedResult
	pr := &parsedResult{
		segments:     make([]string, len(segments)),
		placeholders: make([]placeholder, len(phs)),
	}
	copy(pr.segments, segments)
	copy(pr.placeholders, phs)

	// Store in formatCache
	formatCache.Store(format, pr)
	return pr
}

// ----------------------------------------------------------------------
// The "render" step
// ----------------------------------------------------------------------

func renderSprintf(pr *parsedResult, args []interface{}) string {
	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	defer builderPool.Put(sb)

	// Estimate capacity
	estimated := 0
	for _, seg := range pr.segments {
		estimated += len(seg)
	}
	estimated += len(args) * 10
	sb.Grow(estimated)

	autoIndex := 0
	nPlaceholders := len(pr.placeholders)

	for i := 0; i < nPlaceholders; i++ {
		// Write the literal chunk before this placeholder
		if i < len(pr.segments) {
			sb.WriteString(pr.segments[i])
		}

		ph := &pr.placeholders[i]
		var val interface{}
		switch {
		case ph.PositionalIndex == nil && len(ph.FieldChain) == 0:
			// Automatic
			val = getArgOrNoValue(autoIndex, args)
			autoIndex++
		case ph.PositionalIndex != nil:
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

		if ph.GoFmtVerb == "%v" {
			// No special formatting => handle quickly
			if s, ok := val.(string); ok {
				sb.WriteString(s)
			} else if stringer, ok := val.(fmt.Stringer); ok {
				sb.WriteString(stringer.String())
			} else {
				formatValFast(sb, val)
			}
		} else {
			// We do have a custom spec
			fmt.Fprintf(sb, ph.GoFmtVerb, val)
		}
	}

	// Write leftover segment, if any
	if len(pr.segments) > nPlaceholders {
		sb.WriteString(pr.segments[nPlaceholders])
	}

	return sb.String()
}

// ----------------------------------------------------------------------
// parseFormatFast: same as before but we convert the spec up front
// ----------------------------------------------------------------------

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
				lastSegment.WriteByte('{')
				i += 2
				continue
			}
			// Placeholder
			segments = append(segments, lastSegment.String())
			lastSegment.Reset()

			end := findClosingBraceFast(data[i+1:])
			if end == -1 {
				// No closing => literal
				lastSegment.WriteByte('{')
				i++
				continue
			}
			end += i + 1 // offset

			ph := parsePlaceholderFast(data[i+1 : end])
			placeholders = append(placeholders, ph)

			i = end + 1

		case '}':
			// Check for escaped `}}`
			if i+1 < len(data) && data[i+1] == '}' {
				lastSegment.WriteByte('}')
				i += 2
				continue
			}
			lastSegment.WriteByte('}')
			i++

		default:
			lastSegment.WriteByte(b)
			i++
		}
	}
	if lastSegment.Len() > 0 {
		segments = append(segments, lastSegment.String())
	}
	return segments, placeholders
}

func findClosingBraceFast(data []byte) int {
	for i, b := range data {
		if b == '}' {
			return i
		}
	}
	return -1
}

// parsePlaceholderFast now also directly determines the GoFmtVerb
// so we never have to call placeholderSpecToPrintf at render time.
func parsePlaceholderFast(data []byte) placeholder {
	var ph placeholder

	// Split off spec on first ':'
	specStart := -1
	for i := 0; i < len(data); i++ {
		if data[i] == ':' {
			specStart = i
			break
		}
	}
	var fieldPart []byte
	var spec string

	if specStart >= 0 {
		fieldPart = data[:specStart]
		spec = string(data[specStart+1:])
	} else {
		fieldPart = data
	}

	ph.GoFmtVerb = convertSpecToFmtVerb(spec)

	// Now figure out if fieldPart is a positional index or a field chain
	if len(fieldPart) > 0 {
		// If starts with digit => positional
		if fieldPart[0] >= '0' && fieldPart[0] <= '9' {
			var idx int
			for i := 0; i < len(fieldPart); i++ {
				if fieldPart[i] == '.' {
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
					return ph
				}
			}
			// purely numeric
			idx64, err := strconv.ParseInt(string(fieldPart), 10, 64)
			if err == nil {
				tmp := int(idx64)
				ph.PositionalIndex = &tmp
			}
		} else {
			// named field chain
			ph.FieldChain = strings.Split(string(fieldPart), ".")
		}
	}
	return ph
}

// convertSpecToFmtVerb is an inline version of placeholderSpecToPrintf
func convertSpecToFmtVerb(spec string) string {
	switch spec {
	case "":
		return "%v"
	case "?":
		return "%+v"
	case "x", "X", "o", "O", "b", "B", "d", "c", "U":
		return "%" + spec
	case "e", "E", "f", "F", "g", "G":
		return "%" + spec
	case "s":
		return "%s"
	default:
		return "%" + spec
	}
}

// ----------------------------------------------------------------------
// formatValFast: same as before
// ----------------------------------------------------------------------

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
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			// The "Ivy" hack: look for a "Name" field
			nameField := rv.FieldByName("Name")
			if nameField.IsValid() && nameField.Kind() == reflect.String {
				sb.WriteString(nameField.String())
			} else {
				fmt.Fprintf(sb, "%v", val)
			}
		} else {
			fmt.Fprintf(sb, "%v", val)
		}
	}
}

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

// ----------------------------------------------------------------------
// Field chain lookup
// ----------------------------------------------------------------------

func getFieldChainValueFast(base interface{}, fields []string) interface{} {
	if base == nil || len(fields) == 0 {
		return "<invalid field>"
	}
	val := reflect.ValueOf(base)
	typ := val.Type()

	// pointer chase
	for typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "<invalid field>"
		}
		val = val.Elem()
		typ = val.Type()
	}

	// map at top level?
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

	// struct or nested map
	for i, field := range fields {
		switch val.Kind() {
		case reflect.Struct:
			f := val.FieldByName(field)
			if !f.IsValid() {
				// cached approach
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
		for val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return "<invalid field>"
			}
			val = val.Elem()
		}
		typ = val.Type()
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

		// If it’s a (possibly nested) struct
		if f.Type.Kind() == reflect.Struct {
			buildFieldCache(f.Type, idx, cache)
		} else if f.Type.Kind() == reflect.Ptr && f.Type.Elem().Kind() == reflect.Struct {
			buildFieldCache(f.Type.Elem(), idx, cache)
		}
	}
}

// ----------------------------------------------------------------------
// Arg retrieval
// ----------------------------------------------------------------------

func getArgOrNoValue(idx int, args []interface{}) interface{} {
	if idx < 0 || idx >= len(args) {
		return "<no value>"
	}
	return args[idx]
}
