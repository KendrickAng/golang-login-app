package api

// Request/Response data keys
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

// Login constants
const (
	LOGIN_SUCCESS    = 10
	LOGIN_FAILED     = 11
	EDIT_SUCCESS     = 20
	EDIT_FAILED      = 21
	LOGOUT_SUCCESS   = 30
	INSERT_SUCCESS   = 40
	INSERT_FAILED    = 41
	HOME_SUCCESS     = 50
	HOME_FAILED      = 51
	GET_SESS_SUCCESS = 60
	GET_SESS_FAILED  = 61
)

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
