package security

import (
	"golang.org/x/crypto/bcrypt"
	"log"
)

// Returns the immutable string hash of a password; error is nil if success
func Hash(pw string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		log.Fatalln(err)
	}
	return string(hash)
}

// Checks that a plaintext password hashes to given hash, returns true if equal
func ComparePwHash(pw string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	if err != nil {
		return false // not equal
	}
	return true
}

func ComparePwHashBytes(pw []byte, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, pw)
	if err != nil {
		return false // not equal
	}
	return true
}
