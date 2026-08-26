// Package utils holds small string helpers shared across packages.
package utils

import "strings"

// TrimAll trims whitespace from every element of items.
func TrimAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = strings.TrimSpace(s)
	}
	return out
}
