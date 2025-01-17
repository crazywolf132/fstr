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
	// Reflection approach segments/placeholders
	segments     []string
	placeholders []placeholder

	// If we can do a "native" approach (pure numeric placeholders with no dots & no auto),
	// we store the generated format string. Then we do a direct call to fmt.Sprintf(...).
	// But we still keep placeholders so we can fallback if the user doesn't supply enough args.
	nativeFormat string
	isNative     bool
}

var formatCache sync.Map // map[string]*parsedResult

// ----------------------------------------------------------------------
// Pools
// ----------------------------------------------------------------------

var (
	builderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}
	placeholderPool = sync.Pool{
		New: func() interface{} {
			return make([]placeholder, 0, 8)
		},
	}
	segmentPool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 8)
		},
	}
	fieldCache sync.Map // map[reflect.Type]map[string][]int (for caching struct fields)
)

// ----------------------------------------------------------------------
// Public API
// ----------------------------------------------------------------------

// Sprintf formats according to Rust-like placeholders in `format`,
// including nested struct fields, map lookups, custom format specs, etc.
func Sprintf(format string, args ...interface{}) string {
	if len(format) == 0 {
		return ""
	}
	// Quick check: if no braces, return as-is
	if !strings.ContainsAny(format, "{}") {
		return format
	}

	// Check cache
	if cached, ok := formatCache.Load(format); ok {
		pr := cached.(*parsedResult)
		return render(pr, args)
	}

	// Not cached => parse and store
	pr := parseFormatAndCache(format)
	return render(pr, args)
}

func Printf(format string, args ...interface{}) (int, error) {
	return fmt.Print(Sprintf(format, args...))
}

func Println(format string, args ...interface{}) (int, error) {
	return fmt.Println(Sprintf(format, args...))
}

func Fprintf(w io.Writer, format string, args ...interface{}) (int, error) {
	return fmt.Fprint(w, Sprintf(format, args...))
}

func Fprintln(w io.Writer, format string, args ...interface{}) (int, error) {
	return fmt.Fprintln(w, Sprintf(format, args...))
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

// ----------------------------------------------------------------------
// Placeholder definition
// ----------------------------------------------------------------------

type placeholder struct {
	PositionalIndex *int
	FieldChain      []string
	GoFmtVerb       string // e.g. "%v", "%x", etc.
	IsAuto          bool   // true => "{}"
}

// ----------------------------------------------------------------------
// parseFormatAndCache
// ----------------------------------------------------------------------

func parseFormatAndCache(format string) *parsedResult {
	seg := segmentPool.Get().([]string)
	seg = seg[:0]
	defer segmentPool.Put(seg)

	phs := placeholderPool.Get().([]placeholder)
	phs = phs[:0]
	defer placeholderPool.Put(phs)

	segments, placeholders := parseFormatFast(format, seg, phs)

	pr := &parsedResult{
		segments:     make([]string, len(segments)),
		placeholders: make([]placeholder, len(placeholders)),
	}
	copy(pr.segments, segments)
	copy(pr.placeholders, placeholders)

	// Check if all placeholders are purely numeric (no dots) and not auto => can do native
	allNumericNoDots := true
	for _, ph := range pr.placeholders {
		if ph.IsAuto {
			// e.g. "{}"
			allNumericNoDots = false
			break
		}
		if len(ph.FieldChain) > 0 {
			allNumericNoDots = false
			break
		}
		if ph.PositionalIndex == nil {
			// named field => reflection needed
			allNumericNoDots = false
			break
		}
	}
	if allNumericNoDots {
		// build a single native format string
		nativeFmt := buildNativeFormat(segments, placeholders)
		pr.nativeFormat = nativeFmt
		pr.isNative = true
	}

	formatCache.Store(format, pr)
	return pr
}

// ----------------------------------------------------------------------
// buildNativeFormat: e.g. {1}:{0} => "%[2]v:%[1]v"
// We also handle e.g. "{2:x}" => "%[3]x"
// ----------------------------------------------------------------------

func buildNativeFormat(segments []string, placeholders []placeholder) string {
	var sb strings.Builder
	nph := len(placeholders)
	for i := 0; i < nph; i++ {
		sb.WriteString(segments[i])
		ph := placeholders[i]
		idx := *ph.PositionalIndex + 1 // Go uses 1-based for %[1]v
		if ph.GoFmtVerb == "%v" {
			sb.WriteString(fmt.Sprintf("%%[%d]v", idx))
		} else {
			verb := ph.GoFmtVerb[1:] // skip '%'
			sb.WriteString(fmt.Sprintf("%%[%d]%s", idx, verb))
		}
	}
	if len(segments) > nph {
		sb.WriteString(segments[nph])
	}
	return sb.String()
}

// ----------------------------------------------------------------------
// Rendering
// ----------------------------------------------------------------------

func render(pr *parsedResult, args []interface{}) string {
	if pr.isNative {
		// Check if user provided enough arguments for the highest index
		// If not, we fallback to reflection-based to avoid "%!v(BADINDEX)".
		if haveSufficientArgs(pr.placeholders, args) {
			return fmt.Sprintf(pr.nativeFormat, args...)
		}
	}
	// otherwise reflection approach
	return renderReflection(pr, args)
}

// Find highest positional index among placeholders, check if we have enough args
func haveSufficientArgs(phs []placeholder, args []interface{}) bool {
	maxIndex := -1
	for _, ph := range phs {
		if ph.PositionalIndex != nil && *ph.PositionalIndex > maxIndex {
			maxIndex = *ph.PositionalIndex
		}
	}
	return len(args) > maxIndex
}

// Reflection-based rendering
func renderReflection(pr *parsedResult, args []interface{}) string {
	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	defer builderPool.Put(sb)

	estimated := 0
	for _, seg := range pr.segments {
		estimated += len(seg)
	}
	estimated += len(pr.placeholders) * 10
	sb.Grow(estimated)

	autoIndex := 0
	for i, ph := range pr.placeholders {
		if i < len(pr.segments) {
			sb.WriteString(pr.segments[i])
		}
		var val interface{}
		switch {
		case ph.IsAuto:
			val = getArgOrNoValue(autoIndex, args)
			autoIndex++
		case ph.PositionalIndex != nil:
			val = getArgOrNoValue(*ph.PositionalIndex, args)
			if len(ph.FieldChain) > 0 {
				val = getFieldChainValueFast(val, ph.FieldChain)
			}
		default:
			// e.g. {Name}
			if len(args) > 0 {
				val = getFieldChainValueFast(args[0], ph.FieldChain)
			} else {
				val = "<no value>"
			}
		}

		if ph.GoFmtVerb == "%v" {
			// quick path
			if s, ok := val.(string); ok {
				sb.WriteString(s)
			} else if stringer, ok := val.(fmt.Stringer); ok {
				sb.WriteString(stringer.String())
			} else {
				formatValFast(sb, val)
			}
		} else {
			// custom spec
			fmt.Fprintf(sb, ph.GoFmtVerb, val)
		}
	}

	if len(pr.segments) > len(pr.placeholders) {
		sb.WriteString(pr.segments[len(pr.placeholders)])
	}

	return sb.String()
}

// ----------------------------------------------------------------------
// Parsing
// ----------------------------------------------------------------------

func parseFormatFast(format string, segments []string, placeholders []placeholder) ([]string, []placeholder) {
	data := unsafe.Slice(unsafe.StringData(format), len(format))
	var lastSegment strings.Builder
	lastSegment.Grow(len(data))

	i := 0
	for i < len(data) {
		switch b := data[i]; b {
		case '{':
			// Check for escaped {{
			if i+1 < len(data) && data[i+1] == '{' {
				lastSegment.WriteByte('{')
				i += 2
				continue
			}
			// placeholder
			segments = append(segments, lastSegment.String())
			lastSegment.Reset()

			end := findClosingBrace(data[i+1:])
			if end == -1 {
				// no closing
				lastSegment.WriteByte('{')
				i++
				continue
			}
			end += (i + 1)
			ph := parsePlaceholder(data[i+1 : end])
			placeholders = append(placeholders, ph)
			i = end + 1

		case '}':
			// check for escaped }}
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

func findClosingBrace(data []byte) int {
	for i, b := range data {
		if b == '}' {
			return i
		}
	}
	return -1
}

func parsePlaceholder(data []byte) placeholder {
	var ph placeholder

	if len(data) == 0 {
		// "{}"
		ph.IsAuto = true
		ph.GoFmtVerb = "%v"
		return ph
	}

	// look for ':'
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

	if len(fieldPart) == 0 {
		// means "{}" or ":{spec}"
		ph.IsAuto = true
		return ph
	}

	// if first char is digit => positional
	if fieldPart[0] >= '0' && fieldPart[0] <= '9' {
		// parse until '.' or end
		for i := 0; i < len(fieldPart); i++ {
			if fieldPart[i] == '.' {
				// e.g. "2.City"
				idx64, err := strconv.ParseInt(string(fieldPart[:i]), 10, 64)
				if err == nil {
					tmp := int(idx64)
					ph.PositionalIndex = &tmp
				}
				ph.FieldChain = strings.Split(string(fieldPart[i+1:]), ".")
				return ph
			}
			if fieldPart[i] < '0' || fieldPart[i] > '9' {
				// treat as named => fallback
				ph.FieldChain = strings.Split(string(fieldPart), ".")
				return ph
			}
		}
		// purely numeric => "2"
		idx64, err := strconv.ParseInt(string(fieldPart), 10, 64)
		if err == nil {
			tmp := int(idx64)
			ph.PositionalIndex = &tmp
		}
		return ph
	}

	// named field => e.g. "Name" or "User.Email"
	ph.FieldChain = strings.Split(string(fieldPart), ".")
	return ph
}

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
// Value formatting
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
		// The "Ivy hack": check if struct with a .Name field
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			nameField := rv.FieldByName("Name")
			if nameField.IsValid() && nameField.Kind() == reflect.String {
				sb.WriteString(nameField.String())
				return
			}
			fmt.Fprintf(sb, "%v", val)
		} else {
			fmt.Fprintf(sb, "%v", val)
		}
	}
}

// ----------------------------------------------------------------------
// Manual int/uint printing
// ----------------------------------------------------------------------

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
		buf[length] = byte((i % 10) + '0')
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
		buf[length] = byte((u % 10) + '0')
		length++
		u /= 10
	}
	for j := length - 1; j >= 0; j-- {
		sb.WriteByte(buf[j])
	}
}

// ----------------------------------------------------------------------
// Reflection-based field chain retrieval
// ----------------------------------------------------------------------

func getFieldChainValueFast(base interface{}, fields []string) interface{} {
	if base == nil || len(fields) == 0 {
		return "<invalid field>"
	}
	val := reflect.ValueOf(base)
	typ := val.Type()

	for typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "<invalid field>"
		}
		val = val.Elem()
		typ = val.Type()
	}

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

	for i, field := range fields {
		switch val.Kind() {
		case reflect.Struct:
			fv := val.FieldByName(field)
			if !fv.IsValid() {
				idx, ok := getCachedFieldIndices(typ, field)
				if !ok {
					return "<invalid field>"
				}
				fv = val.FieldByIndex(idx)
				if !fv.IsValid() {
					return "<invalid field>"
				}
			}
			val = fv
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
		if idx, ok2 := cached.(map[string][]int)[fieldName]; ok2 {
			return idx, true
		}
	}
	cache := make(map[string][]int)
	buildFieldCache(typ, nil, cache)
	fieldCache.Store(typ, cache)
	idx, ok := cache[fieldName]
	return idx, ok
}

func buildFieldCache(typ reflect.Type, prefix []int, cache map[string][]int) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		idx := append(prefix, i)
		cache[f.Name] = append([]int(nil), idx...)
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
