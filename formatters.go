package fstr

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	formatters = make(map[reflect.Type]Formatter)
	verbs      = make(map[string]FormattingVerb)
	fmtMutex   sync.RWMutex
)

// FormattingVerb represents a custom formatting verb
type FormattingVerb struct {
	handler func(interface{}, FormatSpecifier) string
	docs    string
}

// RegisterFormatter registers a custom formatter for a specific type
func RegisterFormatter(t reflect.Type, f Formatter) {
	fmtMutex.Lock()
	defer fmtMutex.Unlock()
	formatters[t] = f
}

// RegisterVerb registers a custom formatting verb with documentation
func RegisterVerb(verb string, handler func(interface{}, FormatSpecifier) string, documentation string) {
	fmtMutex.Lock()
	defer fmtMutex.Unlock()
	verbs[verb] = FormattingVerb{
		handler: handler,
		docs:    documentation,
	}
}

// GetFormattingVerb retrieves a registered formatting verb
func GetFormattingVerb(verb string) (FormattingVerb, bool) {
	fmtMutex.RLock()
	defer fmtMutex.RUnlock()
	v, ok := verbs[verb]
	return v, ok
}

// GetFormatter retrieves a custom formatter for a type
func GetFormatter(t reflect.Type) (Formatter, bool) {
	fmtMutex.RLock()
	defer fmtMutex.RUnlock()
	f, ok := formatters[t]
	return f, ok
}

// FormatStruct formats a struct using field names as named arguments
func FormatStruct(s interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(s)

	// Handle pointer to struct
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return result
		}
		val = val.Elem()
	}

	// Ensure we're dealing with a struct
	if val.Kind() != reflect.Struct {
		return result
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get the field tag or use field name
		name := field.Tag.Get("fstr")
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue // Skip fields tagged with "-"
		}

		// Handle embedded structs
		if field.Anonymous && val.Field(i).Kind() == reflect.Struct {
			embedded := FormatStruct(val.Field(i).Interface())
			for k, v := range embedded {
				result[k] = v
			}
			continue
		}

		result[name] = val.Field(i).Interface()
	}

	return result
}

// Default formatters for basic types
type (
	StringFormatter  struct{}
	IntegerFormatter struct{}
	FloatFormatter   struct{}
	BoolFormatter    struct{}
	SliceFormatter   struct{}
	MapFormatter     struct{}
)

// Format implementations for the default formatters
func (sf StringFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	if s, ok := arg.(string); ok {
		return formatString(s, spec)
	}
	return "<invalid>"
}

func (inf IntegerFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	return formatInteger(arg, spec)
}

func (ff FloatFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	return formatFloat(arg, spec)
}

func (bf BoolFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	if b, ok := arg.(bool); ok {
		return formatBool(b, spec)
	}
	return "<invalid>"
}

func (sf SliceFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	val := reflect.ValueOf(arg)
	if val.Kind() != reflect.Slice {
		return "<invalid>"
	}

	var result strings.Builder
	result.WriteString("[")
	for i := 0; i < val.Len(); i++ {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(F("{}", val.Index(i).Interface()))
	}
	result.WriteString("]")
	return result.String()
}

func (mf MapFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	val := reflect.ValueOf(arg)
	if val.Kind() != reflect.Map {
		return "<invalid>"
	}

	var result strings.Builder
	result.WriteString("{")
	for i, key := range val.MapKeys() {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(F("{}: {}", key.Interface(), val.MapIndex(key).Interface()))
	}
	result.WriteString("}")
	return result.String()
}

// Custom verb implementations
func jsonVerb(v interface{}, _ FormatSpecifier) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "<invalid json>"
	}
	return string(data)
}

func hexVerb(v interface{}, _ FormatSpecifier) string {
	return fmt.Sprintf("%x", v)
}

func binVerb(v interface{}, _ FormatSpecifier) string {
	return fmt.Sprintf("%b", v)
}

func upperVerb(v interface{}, _ FormatSpecifier) string {
	if s, ok := v.(string); ok {
		return strings.ToUpper(s)
	}
	return fmt.Sprintf("%v", v)
}

func lowerVerb(v interface{}, _ FormatSpecifier) string {
	if s, ok := v.(string); ok {
		return strings.ToLower(s)
	}
	return fmt.Sprintf("%v", v)
}

// Add TimeFormatter
type TimeFormatter struct{}

func (tf TimeFormatter) Format(arg interface{}, spec FormatSpecifier) string {
	if t, ok := arg.(time.Time); ok {
		switch spec.Type {
		case "":
			return t.Format(time.RFC3339)
		case "date":
			return t.Format("2006-01-02")
		case "time":
			return t.Format("15:04:05")
		default:
			return t.Format(spec.Type)
		}
	}
	return "<invalid time>"
}

// Register in init()
func init() {
	RegisterFormatter(reflect.TypeOf(time.Time{}), TimeFormatter{})
}
