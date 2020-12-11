package main

import (
	"encoding/json"
	"errors"
	"example.com/kendrick/auth"
	"example.com/kendrick/common"
	"example.com/kendrick/http-server/fileio"
	"example.com/kendrick/protocol"
	"example.com/kendrick/security"
	"fmt"
	"html/template"
	"io"
	"log"
	_ "mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	TIMEOUT     = time.Second * 5
	IMG_MAXSIZE = 1 << 12 // 2^12
)

var templates *template.Template

// ********************************
// *********** COMMON *************
// ********************************
func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	file := fmt.Sprintf("%s.html", tmpl)
	err := templates.ExecuteTemplate(w, file, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sendReq(data protocol.Request) net.Conn {
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		log.Fatalln(err)
	}
	req := protocol.CreateRequest(data)
	common.Display("SENDING REQUEST: ", string(req))
	_, err = conn.Write(req)
	if err != nil {
		log.Fatalln(err)
	}
	return conn
}

func receiveRes(w http.ResponseWriter, conn net.Conn) protocol.Response {
	err := conn.SetDeadline(time.Now().Add(TIMEOUT))
	if err != nil {
		log.Panicln(err)
	}
	dec := json.NewDecoder(conn)
	var res protocol.Response
	err = dec.Decode(&res)
	if err == io.EOF {
		common.Display("EOF WHEN READING RESPONSE", nil)
	} else if errors.Is(err, os.ErrDeadlineExceeded) {
		http.Error(w, "TCP Server timeout", http.StatusInternalServerError)
		return protocol.Response{}
	} else if err != nil {
		log.Fatalln(err)
	}
	common.Display("RECEIVED RESPONSE: ", res)
	err = conn.SetDeadline(time.Time{})
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
		Source: "LOGIN",
		Data:   ret,
	}
	common.Display("CREATED LOGIN REQ: ", req)
	return req
}

func processLoginRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Display("PROCESSING LOGIN RES: ", res)
	if res.Code != protocol.USER_FOUND {
		qs := common.QueryString("No such account, please register first!")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	username := res.Data[protocol.Username]
	sid := auth.CreateSession(username)
	http.SetCookie(w, &http.Cookie{
		Name:  auth.SESS_COOKIE_NAME,
		Value: sid,
	})
	qs := common.QueryString("Welcome, " + username)
	http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
}

// Main handler called when logging in
func login(w http.ResponseWriter, r *http.Request) {
	if auth.IsLoggedIn(r) {
		http.Redirect(w, r, "/edit", http.StatusSeeOther)
		return
	}

	req := createLoginReq(r)
	conn := sendReq(req)
	res := receiveRes(w, conn)
	processLoginRes(w, r, res)
	conn.Close()
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "login", desc)
	case http.MethodPost:
		login(w, r)
	default:
		log.Fatalln("Unused method" + r.Method)
	}
}

// ******************************
// *********** EDIT *************
// ******************************
func edit(w http.ResponseWriter, r *http.Request) {
	req, err := createEditReq(r)
	if err != nil {
		common.Display(err.Error())
		http.Redirect(w, r, "/edit"+common.QueryString(err.Error()), http.StatusSeeOther)
		return
	}
	conn := sendReq(req)
	res := receiveRes(w, conn)
	processEditRes(w, r, res)
	conn.Close()
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
	common.Display("FILE INFORMATION: ", header)

	// store image persistently
	username := auth.GetSessionUser(r)
	imgPath := fileio.ImageUpload(file, username)
	common.Display("PROFILE PICTURE UPLOADED: " + imgPath)

	// create return data
	ret := make(map[string]string)
	ret[protocol.Nickname] = nickname
	ret[protocol.ProfilePic] = imgPath
	ret[protocol.Username] = username
	req := protocol.Request{
		Source: "EDIT",
		Data:   ret,
	}
	common.Display("CREATED EDIT REQUEST: ", req)
	return req, nil
}

func processEditRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Display("PROCESSING EDIT RES: ", res)
	switch res.Code {
	case protocol.EDIT_SUCCESS:
		qs := common.QueryString("Edit Success!")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	case protocol.EDIT_FAILED:
		qs := common.QueryString("Edit Failed...")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(r) {
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

// **********************************
// *********** REGISTER *************
// **********************************
func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "register", desc)
	case http.MethodPost:
		create(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func create(w http.ResponseWriter, r *http.Request) {
	req := createRegReq(r)
	conn := sendReq(req)
	res := receiveRes(w, conn)
	processRegRes(w, r, res)
	conn.Close()
}

func createRegReq(r *http.Request) protocol.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	ret := make(map[string]string)
	ret[protocol.Username] = username
	ret[protocol.PwHash] = security.Hash(password)
	ret[protocol.Nickname] = nickname
	req := protocol.Request{
		Source: "REGISTER",
		Data:   ret,
	}
	common.Display("CREATED REGISTER REQ: ", req)
	return req
}

func processRegRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Display("PROCESSING REGISTER RESPONSE: ", res)
	switch res.Code {
	case protocol.INSERT_SUCCESS:
		params := url.Values{
			"desc": {"Account created!"},
		}
		http.Redirect(w, r, "/login?"+params.Encode(), http.StatusSeeOther)
	case protocol.INSERT_FAILED:
		params := url.Values{
			"desc": {"Account creation failed, please try again!"},
		}
		http.Redirect(w, r, "/register?"+params.Encode(), http.StatusSeeOther)
	}
}

// ***************************************
// *********** HTTP HANDLERS *************
// ***************************************
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
	logout(w, r)
}

func logout(w http.ResponseWriter, r *http.Request) {
	req := createLogoutReq(r)
	conn := sendReq(req)
	res := receiveRes(w, conn)
	processLogoutRes(w, r, res)
	conn.Close()
}

func createLogoutReq(r *http.Request) protocol.Request {
	c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
	ret := make(map[string]string)
	ret[protocol.SessionId] = c.Value
	req := protocol.Request{
		Source: "LOGOUT",
		Data:   ret,
	}
	common.Display("CREATED LOGOUT REQUEST: ", req)
	return req
}

func processLogoutRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	common.Display("PROCESSING LOGOUT RESPONSE: ", res)
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

func init() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
}

func main() {
	// have the server listen on required routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/register", registerHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
