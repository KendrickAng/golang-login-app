package security

import (
	"golang.org/x/crypto/bcrypt"
)

// Returns the immutable string hash of a password; error is nil if success
func Hash(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(hash), err
}

// Checks that a password hashes to given hash, returns true if equal
func ComparePwHash(pw string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	if err != nil {
		return false // not equal
	}
	return true
}
