// Package db is a trivial in-memory key-value store.
package db

var data = map[string]string{}

// Get returns the value stored at key, if any.
func Get(key string) (string, bool) {
	v, ok := data[key]
	return v, ok
}

// Set stores value at key.
func Set(key, value string) {
	data[key] = value
}
