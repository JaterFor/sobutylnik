package auth

import "testing"

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		name    string
		pw      string
		wantErr error
	}{
		{"too short", "abc123", ErrPasswordTooShort},
		{"valid", "correct-horse-battery", nil},
		{"too long", stringsRepeat("a", 100), ErrPasswordTooLong},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := ValidatePasswordStrength(c.pw); err != c.wantErr {
				t.Fatalf("got %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !VerifyPassword(hash, "correct-horse-battery") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong-password") {
		t.Fatal("expected wrong password to fail verification")
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
