package auth

import (
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/security"
)

const (
	SESS_COOKIE_NAME     = "session"
	USERNAME_COOKIE_NAME = "username"
)

// valid username-password pairs will be stored here
var validPwCache = make(map[string]string, 250)

/*
This package handles password authentication.
*/

// Checks if a user's password matches the given pw string
func IsValidPassword(user *api.User, pw string) bool {
	// try to get from cache first
	if validPw, ok := validPwCache[user.Username]; ok {
		return validPw == pw
	} else {
		isPwValid := security.ComparePwHash(pw, user.PwHash)
		if isPwValid {
			validPwCache[user.Username] = pw
		}
		return isPwValid
	}
}
