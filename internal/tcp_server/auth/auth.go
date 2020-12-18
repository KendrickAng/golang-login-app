package auth

import (
	"example.com/kendrick/internal/tcp_server/database"
	"example.com/kendrick/internal/tcp_server/security"
	"github.com/satori/uuid"
)

const (
	SESS_COOKIE_NAME = "session"
)

var validPwCache = make(map[string]bool, 250)

/*
This package handles authentication, session creation/deletion, cookie creation/deletion, uuid creation.
It also queries the SQL database, if needed.
*/
func IsValidPassword(username string, pw string) bool {
	users := database.GetUser(username)
	if len(users) != 1 {
		return false
	}
	// try to get from cache first
	if isValid, ok := validPwCache[username]; ok {
		return isValid
	} else {
		isPwValid := security.ComparePwHash(pw, users[0].PwHash)
		validPwCache[username] = isPwValid
		return isPwValid
	}
}

// generates a new uuid for a username, then returns the uuid.
func CreateSession(username string) string {
	id := uuid.NewV4().String()
	database.InsertSession(id, username)
	return id
}

// returns username for a given session uuid
func GetSessionUser(sid string) string {
	users := database.GetSession(sid)
	if len(users) != 1 {
		return ""
	}
	return users[0].Username
}

// Deletes the session associated with this session id
func DelSessionUser(sid string) string {
	username := GetSessionUser(sid)
	database.DeleteSession(sid)
	return username
}
