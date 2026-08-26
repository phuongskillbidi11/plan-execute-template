package main

import "testing"

func TestLastN(t *testing.T) {
	got := LastN([]int{1, 2, 3, 4, 5}, 3)
	want := []int{3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
