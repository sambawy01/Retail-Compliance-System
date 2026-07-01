// Package api — validation.go provides input validation helpers.
package api

import (
	"unicode"
)

// ValidatePassword checks that a password meets complexity requirements.
// Requirements: minimum 8 chars, at least 1 uppercase, 1 lowercase, 1 digit.
// Returns true if valid, false otherwise.
func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// ValidateEmail checks basic email format (contains @ with something before and after).
func ValidateEmail(email string) bool {
	if len(email) < 5 || len(email) > 254 {
		return false
	}
	atIdx := -1
	for i, ch := range email {
		if ch == '@' {
			if atIdx != -1 {
				return false // multiple @
			}
			atIdx = i
		}
	}
	return atIdx > 0 && atIdx < len(email)-1
}