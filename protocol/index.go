package protocol

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"log"
)

const (
	Username   = "username"
	Nickname   = "nickname"
	PwHash     = "pwhash"
	ProfilePic = "profilepic"
)

type User struct {
	Username   string
	Nickname   string
	PwHash     string
	ProfilePic string
}

func Marshall(user User) string {
	bytes := bytes.Buffer{}
	enc := gob.NewEncoder(&bytes)
	err := enc.Encode(user)
	if err != nil {
		log.Panicln("Failed to encode:", err)
	}
	return base64.StdEncoding.EncodeToString(bytes.Bytes())
}

func Unmarshall(str string) User {
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

// Creates a formatted POST request (to be sent to server)
func CreatePost(data map[string]string) {

}

// Registers the user struct for use. Must be called first.
func Init() {
	gob.Register(User{})
}
