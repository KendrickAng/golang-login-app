package auth

import (
	"example.com/kendrick/mysql-db"
	"example.com/kendrick/security"
	"github.com/satori/uuid"
	"log"
	"net/http"
)

const (
	SESS_COOKIE_NAME = "session"
)

var dbSessions = make(map[string]string)

/*
This package handles authentication, session creation/deletion, cookie creation/deletion, uuid creation.
It also queries the SQL database, if needed.
*/
func IsLoggedIn(req *http.Request) bool {
	cookie, err := req.Cookie(SESS_COOKIE_NAME)
	if err != nil {
		return false
	}
	user := dbSessions[cookie.Value]
	return isValidUser(user)
}

// Returns true if the username exists in the SQL DB.
func isValidUser(username string) bool {
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
	dbSessions[id] = username
	log.Println("Session " + id + " Created. DB: ")
	log.Println(dbSessions)
	return id
}

// returns username for a given session uuid, or nil if key doesn't exist
func GetSessionUser(req *http.Request) string {
	cookie, err := req.Cookie(SESS_COOKIE_NAME)
	if err != nil {
		return ""
	}
	return dbSessions[cookie.Value]
}

func DelSessionUser(req *http.Request) {
	cookie, err := req.Cookie(SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Session " + cookie.Value + " Deleted. DB: ")
	log.Println(dbSessions)
	delete(dbSessions, cookie.Value)
}
