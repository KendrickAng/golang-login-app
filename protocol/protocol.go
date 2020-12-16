package protocol

const (
	Username        = "username"
	Nickname        = "nickname"
	PwPlain         = "pw"
	PwHash          = "pwhash"
	ProfilePic      = "profilepic"
	SessionId       = "sid"
	RequestId       = "rid"
	RequestIdHeader = "X-Request-ID"
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
//func EncodeJson(v interface{}) []byte {
//	b, err := json.Marshal(v)
//	if err != nil {
//		log.Panicln(err)
//	}
//	return b
//}

//func Encode(req Request) string {
//	bytes := bytes.Buffer{}
//	err := gob.NewEncoder(&bytes).Encode(req)
//	if err != nil {
//		log.Panicln("Failed to encode:", err)
//	}
//	return base64.StdEncoding.EncodeToString(bytes.Bytes())
//}

//func Decode(res Response) User {
//	u := User{}
//	by, err := base64.StdEncoding.DecodeString(str)
//	if err != nil {
//		log.Panicln("Failed base64 decode: ", err)
//	}
//	b := bytes.Buffer{}
//	b.Write(by)
//	d := gob.NewDecoder(&b)
//	err = d.Decode(&u)
//	if err != nil {
//		log.Panicln("Failed gob decode: ", err)
//	}
//	return u
//}
