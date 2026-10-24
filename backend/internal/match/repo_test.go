package match

import "testing"

func TestOrderedPair(t *testing.T) {
	a, b := orderedPair("user-2", "user-1")
	if a != "user-1" || b != "user-2" {
		t.Fatalf("got (%q, %q), want (user-1, user-2)", a, b)
	}

	a2, b2 := orderedPair("user-1", "user-2")
	if a2 != "user-1" || b2 != "user-2" {
		t.Fatalf("got (%q, %q), want (user-1, user-2)", a2, b2)
	}
}
