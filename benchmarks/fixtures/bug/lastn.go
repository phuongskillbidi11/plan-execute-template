package main

// LastN returns the last n elements of items.
func LastN(items []int, n int) []int {
	if n > len(items) {
		n = len(items)
	}
	return items[len(items)-n-1:] // BUG: off-by-one, should be len(items)-n
}

func main() {}
