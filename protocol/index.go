package protocol

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
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

type Request struct {
	Source string
	Data   map[string]string
}

type Response struct {
	Code        int
	Description string
	Data        map[string]string
}

// Login constants
const (
	NO_SUCH_USER   = 100
	USER_FOUND     = 101
	EDIT_SUCCESS   = 200
	EDIT_FAILED    = 201
	LOGOUT_SUCCESS = 300
	INSERT_SUCCESS = 400
	INSERT_FAILED  = 401
)

// Creates an encoded POST request (to be sent to server)
func CreateRequest(req Request) []byte {
	b, err := json.Marshal(req)
	if err != nil {
		log.Panicln(err)
	}
	return b
}

func CreateResponse(res Response) []byte {
	b, err := json.Marshal(res)
	if err != nil {
		log.Panicln(err)
	}
	return b
}

func encode(user User) string {
	bytes := bytes.Buffer{}
	enc := gob.NewEncoder(&bytes)
	err := enc.Encode(user)
	if err != nil {
		log.Panicln("Failed to encode:", err)
	}
	return base64.StdEncoding.EncodeToString(bytes.Bytes())
}

func decode(str string) User {
	u := User{}
	by, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		log.Panicln("Failed base64 decode: ", err)
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&u)
	if err != nil {
		log.Panicln("Failed gob decode: ", err)
	}
	return u
}

// Registers the user struct for use. Must be called first.
func Init() {
	gob.Register(User{})
}
