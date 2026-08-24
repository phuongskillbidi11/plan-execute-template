package main

import (
	"reflect"
	"testing"
)

// TestReorderFlagsFirst guards against a real defect found during Phase 3
// integration testing: Go's flag package stops parsing at the first
// non-flag token, so "eng plan review <dir> --verdict PASS" silently
// discarded --verdict entirely (the review verdict was never recorded).
func TestReorderFlagsFirstPositionalBeforeFlag(t *testing.T) {
	got := reorderFlagsFirst([]string{"some-dir", "--verdict", "PASS"}, map[string]bool{})
	want := []string{"--verdict", "PASS", "some-dir"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReorderFlagsFirstFlagBeforePositional(t *testing.T) {
	got := reorderFlagsFirst([]string{"--risk", "high-risk", "my-plan"}, map[string]bool{})
	want := []string{"--risk", "high-risk", "my-plan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReorderFlagsFirstBoolFlagDoesNotConsumeNextToken(t *testing.T) {
	got := reorderFlagsFirst([]string{"my-plan", "--requires-approval"}, map[string]bool{"requires-approval": true})
	want := []string{"--requires-approval", "my-plan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestReorderFlagsFirstEqualsForm(t *testing.T) {
	got := reorderFlagsFirst([]string{"my-plan", "--reason=broken"}, map[string]bool{})
	want := []string{"--reason=broken", "my-plan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
