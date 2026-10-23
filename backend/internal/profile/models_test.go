package profile

import "testing"

func TestIsValidGender(t *testing.T) {
	cases := map[string]bool{"male": true, "female": true, "other": true, "alien": false, "": false}
	for g, want := range cases {
		if got := IsValidGender(g); got != want {
			t.Errorf("IsValidGender(%q) = %v, want %v", g, got, want)
		}
	}
}

func TestIsValidReadinessStatus(t *testing.T) {
	cases := map[string]bool{"now": true, "today": true, "this_week": true, "tomorrow": false, "": false}
	for s, want := range cases {
		if got := IsValidReadinessStatus(s); got != want {
			t.Errorf("IsValidReadinessStatus(%q) = %v, want %v", s, got, want)
		}
	}
}
