package api

const (
	Username        = "username"
	Nickname        = "nickname"
	PwPlain         = "pw"
	PwHash          = "pwhash"
	ProfilePic      = "profilepic"
	SessionId       = "sid"
	RequestId       = "rid"
	RequestIdHeader = "X-Request-ID"
	ResCode         = "resCode"
	ResDesc         = "ResDesc"
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
	Id   string // uuid for logging
	Type string
	Data map[string]string
}

type Response struct {
	Id          string //uuid for logging
	Code        int
	Description string
	Data        map[string]string
}

// Login constants
const (
	CREDENTIALS_VALID   = 10
	CREDENTIALS_INVALID = 11
	EDIT_SUCCESS        = 20
	EDIT_FAILED         = 21
	LOGOUT_SUCCESS      = 30
	INSERT_SUCCESS      = 40
	INSERT_FAILED       = 41
)
