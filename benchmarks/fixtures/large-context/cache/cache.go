// Package cache is a minimal fixed-size cache stub.
package cache

var entries = map[string]string{}

// Lookup returns a cached value, if present.
func Lookup(key string) (string, bool) {
	v, ok := entries[key]
	return v, ok
}

// Store caches a value.
func Store(key, value string) {
	entries[key] = value
}
