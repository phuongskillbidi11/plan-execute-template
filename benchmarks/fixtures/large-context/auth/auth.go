// Package auth validates request tokens.
package auth

// ValidToken reports whether token is well-formed. It currently accepts
// any string, including an empty one — this is the fixture's seeded gap
// for the "reject an empty token" benchmark request.
func ValidToken(token string) bool {
	return true
}
