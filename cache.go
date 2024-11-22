package fstr

import (
	"reflect"
	"strings"
	"sync"
)

// formatCache holds parsed format strings for reuse
type formatCache struct {
	cache map[string]parsedFormat
	mu    sync.RWMutex
}

type parsedFormat struct {
	literals     []string
	placeholders []placeholder
	accessPaths  map[string][]string // For nested field access
}

var (
	globalCache = &formatCache{
		cache: make(map[string]parsedFormat),
	}
)

// Add these constants
const (
	maxCacheSize = 1000 // Maximum number of cached formats
	cleanupRatio = 0.5  // Remove this portion of cache when limit is reached
)

// getParsedFormat returns a cached parsed format or parses and caches a new one
func getParsedFormat(format string) parsedFormat {
	// Try to get from cache first
	globalCache.mu.RLock()
	if pf, ok := globalCache.cache[format]; ok {
		globalCache.mu.RUnlock()
		return pf
	}
	globalCache.mu.RUnlock()

	// Parse the format string
	literals, placeholders := parse(format)

	// Build access paths for nested fields
	accessPaths := make(map[string][]string)
	for _, ph := range placeholders {
		if ph.name != "" {
			paths := strings.Split(ph.name, ".")
			if len(paths) > 1 {
				accessPaths[ph.name] = paths
			}
		}
	}

	// Cache the result
	pf := parsedFormat{
		literals:     literals,
		placeholders: placeholders,
		accessPaths:  accessPaths,
	}

	globalCache.mu.Lock()
	globalCache.cache[format] = pf
	globalCache.mu.Unlock()

	return pf
}

// getNestedValue retrieves a value from a nested structure using dot notation
func getNestedValue(data interface{}, path []string) interface{} {
	current := data
	for _, key := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			if val, ok := v[key]; ok {
				current = val
			} else {
				return nil
			}
		default:
			val := reflect.ValueOf(current)
			if val.Kind() == reflect.Ptr {
				val = val.Elem()
			}
			if val.Kind() != reflect.Struct {
				return nil
			}
			field := val.FieldByName(key)
			if !field.IsValid() {
				return nil
			}
			current = field.Interface()
		}
	}
	return current
}

// clearCache clears the format cache
func clearCache() {
	globalCache.mu.Lock()
	globalCache.cache = make(map[string]parsedFormat)
	globalCache.mu.Unlock()
}

// Add this method to formatCache
func (c *formatCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) > maxCacheSize {
		// Create new cache with reduced size
		newSize := int(float64(maxCacheSize) * cleanupRatio)
		newCache := make(map[string]parsedFormat, newSize)

		// Keep most recently used formats
		i := 0
		for k, v := range c.cache {
			if i >= newSize {
				break
			}
			newCache[k] = v
			i++
		}
		c.cache = newCache
	}
}
