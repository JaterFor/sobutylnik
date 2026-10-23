package auth

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8
const maxPasswordLength = 72 // bcrypt limit

var ErrPasswordTooShort = errors.New("password must be at least 8 characters")
var ErrPasswordTooLong = errors.New("password must be at most 72 characters")

func ValidatePasswordStrength(password string) error {
	n := utf8.RuneCountInString(password)
	if n < minPasswordLength {
		return ErrPasswordTooShort
	}
	if n > maxPasswordLength {
		return ErrPasswordTooLong
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
