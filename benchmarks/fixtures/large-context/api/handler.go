// Package api handles inbound HTTP requests and dispatches them to the
// appropriate store/cache/auth package.
package api

import "fmt"

// Handle processes a request path and returns a canned response.
func Handle(path string) string {
	return fmt.Sprintf("handled: %s", path)
}
