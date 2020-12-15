package auth

import (
	"example.com/kendrick/database"
	"example.com/kendrick/security"
	"github.com/satori/uuid"
)

const (
	SESS_COOKIE_NAME = "session"
)

/*
This package handles authentication, session creation/deletion, cookie creation/deletion, uuid creation.
It also queries the SQL database, if needed.
*/
func IsLoggedIn(uuid string) bool {
	return uuid != ""
	//if len(uuid) == 0 {
	//	return false
	//}
	//users := database.GetSession(uuid)
	//if len(users) != 1 {
	//	return false
	//}
	//return isValidUser(users[0].Username)
}

// Returns true if the username exists in the SQL DB.
func isValidUser(username string) bool {
	if len(username) == 0 {
		return false
	}
	users := database.GetUser(username)
	return len(users) == 1
}

func IsValidPassword(username string, pw string) bool {
	users := database.GetUser(username)
	if len(users) != 1 {
		return false
	}
	return security.ComparePwHash(pw, users[0].PwHash)
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
