package protocol

import (
	"encoding/json"
	"log"
)

const (
	Username   = "username"
	Nickname   = "nickname"
	PwPlain    = "password"
	PwHash     = "pwhash"
	ProfilePic = "profilepic"
	SessionId  = "sessionid"
)

type User struct {
	Username   string
	Nickname   string
	PwHash     string
	ProfilePic string
}

type Session struct {
	Uuid     string
	Username string
}

type Request struct {
	Type string
	Data map[string]string
}

type Response struct {
	Code        int
	Description string
	Data        map[string]string
}

// Login constants
const (
	CREDENTIALS_VALID     = 10
	CREDENTIALS_INVALID   = 11
	EDIT_SUCCESS          = 20
	EDIT_FAILED           = 21
	LOGOUT_SUCCESS        = 30
	INSERT_SUCCESS        = 40
	INSERT_FAILED         = 41
	TCP_CONNECTION_CLOSED = 444
	TCP_SERVER_TIMEOUT    = 504
)

// Creates an encoded POST request (to be sent to server)
func EncodeJson(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Panicln(err)
	}
	return b
}
