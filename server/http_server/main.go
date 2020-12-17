package main

import (
	"encoding/gob"
	"errors"
	"example.com/kendrick/auth"
	"example.com/kendrick/common"
	"example.com/kendrick/fileio"
	"example.com/kendrick/protocol"
	"example.com/kendrick/security"
	"fmt"
	"github.com/satori/uuid"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io"
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
	LOG_LEVEL   = log.DebugLevel
)

type HTTPServer struct {
	Port string
}

var templates *template.Template
var logger *log.Logger

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

func handleError(rid string, conn net.Conn, err error) {
	if err != nil {
		logger := log.WithFields(log.Fields{protocol.RequestId: rid})
		if err == io.EOF {
			// connection closed by TCP server
			logger.Error(err)
			conn.Close()
		} else if errors.Is(err, os.ErrDeadlineExceeded) {
			// read deadline exceeded, do nothing
			logger.Error(err)
		} else {
			logger.Error("Others: ", err)
		}
	}
}

func getTcpConn() net.Conn {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		log.Error(err)
	}
	return conn
}

func sendReq(conn net.Conn, data protocol.Request) error {
	//conn, err := net.Dial("tcp", "localhost:8081")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	log.Debug("Sending request", data)
	err := gob.NewEncoder(conn).Encode(&data)
	if err != nil {
		return err
	}
	log.Debug("Request sent", data)
	return nil
}

func receiveRes(conn net.Conn) (protocol.Response, error) {
	log.Debug("Receiving response")
	err := conn.SetReadDeadline(time.Now().Add(TIMEOUT))
	if err != nil {
		return protocol.Response{}, err
	}
	var res protocol.Response
	err = gob.NewDecoder(conn).Decode(&res)
	if err != nil {
		return protocol.Response{}, err
	}
	err = conn.SetReadDeadline(time.Time{})
	if err != nil {
		return protocol.Response{}, err
	}
	log.Debug("Received response", res)
	//if err == io.EOF {
	//	common.Print("EOF WHEN READING RESPONSE", nil)
	//	return protocol.Response{
	//		Code:        protocol.TCP_CONNECTION_CLOSED,
	//		Description: "TCP Server closed connection, EOF when reading response",
	//		Data:        nil,
	//	}
	//} else if errors.Is(err, os.ErrDeadlineExceeded) {
	//	return protocol.Response{
	//		Code:        protocol.TCP_SERVER_TIMEOUT,
	//		Description: "TCP Server timeout after " + strconv.FormatInt(int64(TIMEOUT), 10) + " seconds",
	//		Data:        nil,
	//	}
	//} else {
	//	handleError(err)
	//}
	return res, nil
}

// *******************************
// *********** LOGIN *************
// *******************************
func createLoginReq(r *http.Request) protocol.Request {
	log.Debug("Creating login request")
	username := r.FormValue("username")
	password := r.FormValue("password")
	rid := r.Header.Get(protocol.RequestIdHeader)
	ret := make(map[string]string)
	ret[protocol.Username] = username
	ret[protocol.PwPlain] = password
	req := protocol.Request{
		Id:   rid,
		Type: "LOGIN",
		Data: ret,
	}
	log.WithFields(log.Fields{
		protocol.RequestId: rid,
		protocol.Username:  username,
		protocol.PwPlain:   password,
	}).Debug("Created login request")
	return req
}

func processLoginRes(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	logger := log.WithField(protocol.RequestId, res.Id)
	logger.Debug("Processing login response")
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
	logger.Debug("Processed login response")
	return
}

// Main handler called when logging in
func login(w http.ResponseWriter, r *http.Request) {
	if auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	req := createLoginReq(r)
	log.Info("Create login request", req)
	conn := getTcpConn()
	defer conn.Close()
	err := sendReq(conn, req)
	if err != nil {
		log.Info("In request")
		handleError(req.Id, conn, err)
		qs := common.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	res, err := receiveRes(conn)
	log.Info("Receive login response", res)
	if err != nil {
		log.Info("In response")
		handleError(req.Id, conn, err)
		qs := common.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	processLoginRes(w, r, res)
	log.WithField(protocol.RequestId, req.Id).Info("Connection closed")
}

// ******************************
// *********** EDIT *************
// ******************************
//func edit(w http.ResponseWriter, r *http.Request) {
//	req, err := createEditReq(r)
//	if err != nil {
//		log.Println(err)
//		http.Redirect(w, r, "/edit"+common.CreateQueryString(err.Error()), http.StatusSeeOther)
//		return
//	}
//	conn := getTcpConn()
//	defer conn.Close()
//	err = sendReq(conn, req)
//	if err != nil {
//		handleError(conn, err)
//		qs := common.CreateQueryString("Edit failed, please try again in a while")
//		http.Redirect(w, r, "/edit" + qs, http.StatusSeeOther)
//		return
//	}
//	res, err := receiveRes(conn)
//	if err != nil {
//		handleError(conn, err)
//		qs := common.CreateQueryString("Edit failed, please try again in a while")
//		http.Redirect(w, r, "/edit" + qs, http.StatusSeeOther)
//		return
//	}
//	processEditRes(w, r, res)
//}

func createEditReq(r *http.Request) (protocol.Request, error) {
	// retrieve form values
	nickname := r.FormValue("nickname")
	file, header, err := r.FormFile("pic")
	if err != nil {
		log.Error(err)
	}
	// enforce max size
	if header.Size > IMG_MAXSIZE {
		err := errors.New("Image too large: maximum " + strconv.Itoa(IMG_MAXSIZE) + " bytes.")
		return protocol.Request{}, err
	}
	defer file.Close()

	// store image persistently
	cookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	imgPath := fileio.ImageUpload(file, auth.GetSessionUser(cookie.Value))

	// create return data
	ret := make(map[string]string)
	ret[protocol.Nickname] = nickname
	ret[protocol.ProfilePic] = imgPath
	ret[protocol.SessionId] = cookie.Value
	req := protocol.Request{
		Type: "EDIT",
		Data: ret,
	}
	log.Debug("CREATED EDIT REQUEST: ", req)
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
//func registerUser(w http.ResponseWriter, r *http.Request) {
//	req := createRegisterReq(r)
//	conn := getTcpConn()
//	defer conn.Close()
//	// TODO
//	_ = sendReq(conn, req)
//	res, _ := receiveRes(conn)
//	processRegisterRes(w, r, res)
//}

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
//func logout(w http.ResponseWriter, r *http.Request) {
//	req := createLogoutReq(r)
//	conn := getTcpConn()
//	defer conn.Close()
//	// TODO
//	_ = sendReq(conn, req)
//	res, _ := receiveRes(conn)
//	processLogoutRes(w, r, res)
//}

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
//func home(w http.ResponseWriter, r *http.Request) {
//	req := createHomeReq(r)
//	conn := getTcpConn()
//	defer conn.Close()
//	// TODO
//	_ = sendReq(conn, req)
//	res, _ := receiveRes(conn)
//	processHomeRes(w, r, res)
//}

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
		//edit(w, r)
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
		//registerUser(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	//home(w, r)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !auth.IsLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	//logout(w, r)
}

func initLogger() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "Jan _2 15:04:05.000000"
	customFormatter.FullTimestamp = true
	customFormatter.ForceColors = false
	customFormatter.DisableColors = true
	//logger = log.New()
	log.SetFormatter(customFormatter)
	err := os.Remove("http.txt")
	if err != nil {
		log.Println(err)
	}
	file, err := os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Println(err)
	}
	//log.SetOutput(file)
	log.SetOutput(io.MultiWriter(file, os.Stdout))
	log.SetLevel(LOG_LEVEL)
}

func withRequestId(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(protocol.RequestIdHeader)
		if rid == "" {
			rid = uuid.NewV4().String()
			r.Header.Set(protocol.RequestIdHeader, rid)
		}
		handler(w, r)
	}
}

func (srv *HTTPServer) Start() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
	//database.Connect()
	//profiling.InitLogFiles()
	initLogger()

	log.Info("HTTP server listening on port ", srv.Port)

	// have the server listen on required routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/login", withRequestId(loginHandler))
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/home", homeHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/register", registerHandler)
	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", srv.Port), nil))
}

func (srv *HTTPServer) Stop() {
	log.Info("HTTP server stopped.")
}

func main() {
	server := HTTPServer{Port: "8080"}
	defer server.Stop()
	server.Start()
}
