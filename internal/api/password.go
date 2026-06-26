// Package api — password.go provides bcrypt password hashing.
package api

import "golang.org/x/crypto/bcrypt"

// bcryptHash hashes a password using bcrypt with cost 12.
func bcryptHash(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, 12)
}

// bcryptCompare compares a bcrypt hash against a plaintext password.
// Returns nil on match, error on mismatch.
func bcryptCompare(hash, password []byte) error {
	return bcrypt.CompareHashAndPassword(hash, password)
}
