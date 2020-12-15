package main

import (
	"encoding/gob"
	"errors"
	"example.com/kendrick/auth"
	"example.com/kendrick/common"
	"example.com/kendrick/fileio"
	"example.com/kendrick/profiling"
	"example.com/kendrick/protocol"
	"example.com/kendrick/security"
	"fmt"
	"html/template"
	"io"
	"log"
	_ "mime/multipart"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	TIMEOUT     = time.Second * 7
	IMG_MAXSIZE = 1 << 12 // 2^12
)

var templates *template.Template

// ********************************
// *********** COMMON *************
// ********************************
// Gets the value of the session cookie. Returns "" if not present.
func getSid(req *http.Request) string {
	cookie, err := req.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil && errors.Is(err, http.ErrNoCookie) {
		return ""
	}
	return cookie.Value
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	file := fmt.Sprintf("%s.html", tmpl)
	err := templates.ExecuteTemplate(w, file, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}

func sendReq(data protocol.Request) net.Conn {
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		log.Fatalln(err)
	}
	err = gob.NewEncoder(conn).Encode(&data)
	common.Print("SENDING REQUEST: ", data)
	if err != nil {
		log.Println(err)
	}
	return conn
}

func receiveRes(conn net.Conn) protocol.Response {
	err := conn.SetDeadline(time.Now().Add(TIMEOUT))
	handleError(err)
	var res protocol.Response
	err = gob.NewDecoder(conn).Decode(&res)
	if err == io.EOF {
		common.Print("EOF WHEN READING RESPONSE", nil)
		return protocol.Response{
			Code:        protocol.TCP_CONNECTION_CLOSED,
			Description: "TCP Server closed connection, EOF when reading response",
			Data:        nil,
		}
	} else if errors.Is(err, os.ErrDeadlineExceeded) {
		return protocol.Response{
			Code:        protocol.TCP_SERVER_TIMEOUT,
			Description: "TCP Server timeout after " + strconv.FormatInt(int64(TIMEOUT), 10) + " seconds",
			Data:        nil,
		}
	} else {
		handleError(err)
	}
	common.Print("RECEIVED RESPONSE: ", res)
	//err = conn.SetDeadline(time.Time{})
	return res
}

// *******************************
// *********** LOGIN *************
// *******************************
func createLoginReq(r *http.Request) protocol.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	ret := make(map[string]string)
	ret[protocol.Username] = username
	ret[protocol.PwPlain] = password
	req := protocol.Request{
		Type: "LOGIN",
		Data: ret,
	}
	common.Print("CREATED LOGIN REQ: ", req)
	return req
}

func processLoginRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Print("PROCESSING LOGIN RES: ", res)
	if res.Code != protocol.CREDENTIALS_INVALID {
		qs := common.CreateQueryString("No such account, please register first!")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	sid := res.Data[protocol.SessionId]
	http.SetCookie(w, &http.Cookie{
		Name:  auth.SESS_COOKIE_NAME,
		Value: sid,
	})
	http.Redirect(w, r, "/home", http.StatusSeeOther)
	return
}

// Main handler called when logging in
func login(w http.ResponseWriter, r *http.Request) {
	if auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}

	req := createLoginReq(r)
	conn := sendReq(req)
	defer conn.Close()
	res := receiveRes(conn)
	processLoginRes(w, r, res)
}

// ******************************
// *********** EDIT *************
// ******************************
func edit(w http.ResponseWriter, r *http.Request) {
	req, err := createEditReq(r)
	if err != nil {
		common.Print(err.Error())
		http.Redirect(w, r, "/edit"+common.CreateQueryString(err.Error()), http.StatusSeeOther)
		return
	}
	conn := sendReq(req)
	defer conn.Close()
	res := receiveRes(conn)
	processEditRes(w, r, res)
}

func createEditReq(r *http.Request) (protocol.Request, error) {
	// retrieve form values
	nickname := r.FormValue("nickname")
	file, header, err := r.FormFile("pic")
	if err != nil {
		log.Fatalln(err)
	}
	// enforce max size
	if header.Size > IMG_MAXSIZE {
		err := errors.New("Image too large: maximum " + strconv.Itoa(IMG_MAXSIZE) + " bytes.")
		return protocol.Request{}, err
	}
	defer file.Close()
	common.Print("FILE INFORMATION: ", header)

	// store image persistently
	cookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	imgPath := fileio.ImageUpload(file, auth.GetSessionUser(cookie.Value))
	common.Print("PROFILE PICTURE UPLOADED: " + imgPath)

	// create return data
	ret := make(map[string]string)
	ret[protocol.Nickname] = nickname
	ret[protocol.ProfilePic] = imgPath
	ret[protocol.SessionId] = cookie.Value
	req := protocol.Request{
		Type: "EDIT",
		Data: ret,
	}
	common.Print("CREATED EDIT REQUEST: ", req)
	return req, nil
}

func processEditRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Print("PROCESSING EDIT RES: ", res)
	switch res.Code {
	case protocol.EDIT_SUCCESS:
		qs := common.CreateQueryString("Edit Success!")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	case protocol.EDIT_FAILED:
		qs := common.CreateQueryString("Edit Failed...")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	}
}

// **********************************
// *********** REGISTER *************
// **********************************
func registerUser(w http.ResponseWriter, r *http.Request) {
	req := createRegisterReq(r)
	conn := sendReq(req)
	res := receiveRes(conn)
	processRegisterRes(w, r, res)
	conn.Close()
}

func createRegisterReq(r *http.Request) protocol.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	ret := make(map[string]string)
	ret[protocol.Username] = username
	ret[protocol.PwHash] = security.Hash(password)
	ret[protocol.Nickname] = nickname
	req := protocol.Request{
		Type: "REGISTER",
		Data: ret,
	}
	common.Print("CREATED REGISTER REQ: ", req)
	return req
}

func processRegisterRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Print("PROCESSING REGISTER RESPONSE: ", res)
	switch res.Code {
	case protocol.INSERT_SUCCESS:
		qs := common.CreateQueryString("Account created!")
		http.Redirect(w, r, "/login?"+qs, http.StatusSeeOther)
	case protocol.INSERT_FAILED:
		params := url.Values{
			"desc": {"Account creation failed, please try again!"},
		}
		http.Redirect(w, r, "/register?"+params.Encode(), http.StatusSeeOther)
	}
}

// ********************************
// *********** LOGOUT *************
// ********************************
func logout(w http.ResponseWriter, r *http.Request) {
	req := createLogoutReq(r)
	conn := sendReq(req)
	res := receiveRes(conn)
	processLogoutRes(w, r, res)
	conn.Close()
}

func createLogoutReq(r *http.Request) protocol.Request {
	c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
	ret := make(map[string]string)
	ret[protocol.SessionId] = c.Value
	req := protocol.Request{
		Type: "LOGOUT",
		Data: ret,
	}
	common.Print("CREATED LOGOUT REQUEST: ", req)
	return req
}

func processLogoutRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Print("PROCESSING LOGOUT RESPONSE: ", res)
	switch res.Code {
	case protocol.LOGOUT_SUCCESS:
		// delete cookie
		c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
		c = &http.Cookie{
			Name:   auth.SESS_COOKIE_NAME,
			Value:  "",
			MaxAge: -1,
		}
		http.SetCookie(w, c)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// ******************************
// *********** HOME *************
// ******************************
func home(w http.ResponseWriter, r *http.Request) {
	req := createHomeReq(r)
	conn := sendReq(req)
	defer conn.Close()
	res := receiveRes(conn)
	processHomeRes(w, r, res)
}

// retrieves the current user based on session cookie.
func createHomeReq(r *http.Request) protocol.Request {
	cookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	// get the user details of this session id
	data := make(map[string]string)
	data[protocol.SessionId] = cookie.Value
	req := protocol.Request{
		Type: "HOME",
		Data: data,
	}
	common.Print("CREATED HOME REQUEST: ", req)
	return req
}

func processHomeRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Print("PROCESSING HOME RESPONSE: ", res)
	switch res.Code {
	case protocol.CREDENTIALS_INVALID:
		renderTemplate(w, "home", res.Data)
	case protocol.CREDENTIALS_VALID:
		qs := common.CreateQueryString("User not found, please login!")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
	}
}

// ***************************************
// *********** HTTP HANDLERS *************
// ***************************************
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "login", desc)
	case http.MethodPost:
		// profiling.RecordLogin("LOGIN")
		login(w, r)
	default:
		log.Fatalln("Unused method" + r.Method)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/login", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "edit", desc)
	case http.MethodPost:
		edit(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "register", desc)
	case http.MethodPost:
		registerUser(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	home(w, r)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	logout(w, r)
}

func init() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
	// database.Connect()
	profiling.InitLogFiles()
}

func main() {
	// have the server listen on required routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/home", homeHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/register", registerHandler)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
