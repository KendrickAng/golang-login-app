package main

import (
	"errors"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/http_server/pool"
	"example.com/kendrick/internal/tcp_server/auth"
	"example.com/kendrick/internal/tcp_server/security"
	"example.com/kendrick/internal/utils"
	"flag"
	"fmt"
	"github.com/satori/uuid"
	poolSP "github.com/silenceper/pool"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io"
	"io/ioutil"
	_ "mime/multipart"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"
)

var (
	logOutput = flag.String(
		"logOutput",
		"",
		"Logrus log output, NONE/FILE/STDERR/ALL, default: STDERR",
	)
	logLevel = flag.String("logLevel", "", "Logrus log level, DEBUG/ERROR/INFO, default: INFO")
)

const IMG_MAXSIZE = 1 << 12 // 2^12

type HTTPServer struct {
	Server   http.Server
	TcpPool  pool.Pool
	Hostname string
	Port     string
}

var templates *template.Template
var logger *log.Logger

// ********************************
// *********** COMMON *************
// ********************************
func isLoggedIn(sid string) bool {
	return sid != ""
}

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

func (srv *HTTPServer) handleError(rid string, conn *pool.TcpConn, err error) {
	if err != nil {
		logger := log.WithFields(log.Fields{api.RequestId: rid})
		if err == io.EOF {
			// connection closed by TCP server - tell the pool to destroy connection
			logger.Error(err)
			err := srv.TcpPool.Destroy(conn)
			if err != nil {
				logger.Error(err)
			}
			//srv.TcpPool.Close(conn)
			//if err != nil {
			//	logger.Error(err)
			//}
			//conn.Close()
		} else if errors.Is(err, os.ErrDeadlineExceeded) {
			// read deadline exceeded, do nothing
			logger.Error(err)
		} else {
			logger.Error("Others: ", err)
		}
	}
}

func (srv *HTTPServer) getTcpConnPooled() (pool.TcpConn, error) {
	conn, err := srv.TcpPool.Get()
	if err != nil {
		return pool.TcpConn{}, err
	}
	return conn, nil
}

func (srv *HTTPServer) getTcpConn() (net.Conn, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		return nil, err
	}
	return conn, err
}

// *******************************
// *********** LOGIN *************
// *******************************
// Main handler called when logging in
func (srv *HTTPServer) login(w http.ResponseWriter, r *http.Request) {
	if isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	req := createLoginReq(r)
	log.Info("Create login request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive login response", res)

	// PROCESS RESPONSE
	processLoginRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createLoginReq(r *http.Request) api.Request {
	log.Debug("Creating login request")
	username := r.FormValue("username")
	password := r.FormValue("password")
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.Username] = username
	ret[api.PwPlain] = password
	req := api.Request{
		Id:   rid,
		Type: "LOGIN",
		Data: ret,
	}
	log.WithFields(log.Fields{
		api.RequestId: rid,
		api.Username:  username,
		api.PwPlain:   password,
	}).Debug("Created login request")
	return req
}

func processLoginRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	logger := log.WithFields(log.Fields{
		api.RequestId: res.Id,
		api.ResCode:   res.Code,
		api.ResDesc:   res.Description,
	})
	logger.Debug("Processing login response")
	if res.Code == api.CREDENTIALS_INVALID {
		qs := utils.CreateQueryString("No such account, please register first!")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	sid := res.Data[api.SessionId]
	username := res.Data[api.Username]
	http.SetCookie(w, &http.Cookie{
		Name:  auth.SESS_COOKIE_NAME,
		Value: sid,
	})
	http.SetCookie(w, &http.Cookie{
		Name:  auth.USERNAME_COOKIE_NAME,
		Value: username,
	})
	http.Redirect(w, r, "/home", http.StatusSeeOther)
	logger.Debug("Processed login response")
	return
}

// ******************************
// *********** EDIT *************
// ******************************
func (srv *HTTPServer) edit(w http.ResponseWriter, r *http.Request) {
	req, err := createEditReq(r)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/edit"+utils.CreateQueryString(err.Error()), http.StatusSeeOther)
		return
	}
	log.Info("Create edit request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Edit failed, please try again later")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Edit failed, please try again in a while")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Edit failed, please try again in a while")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive edit response", res)

	// PROCESS RESPONSE
	processEditRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createEditReq(r *http.Request) (api.Request, error) {
	// retrieve form values
	nickname := r.FormValue("nickname")
	file, header, err := r.FormFile("pic")
	if err != nil {
		log.Error(err)
	}
	// enforce max size
	if header.Size > IMG_MAXSIZE {
		err := errors.New("Image too large: maximum " + strconv.Itoa(IMG_MAXSIZE) + " bytes.")
		return api.Request{}, err
	}
	defer file.Close()

	// store image persistently
	unameCookie, err := r.Cookie(auth.USERNAME_COOKIE_NAME)
	if err != nil {
		log.Error(err)
		return api.Request{}, err
	}
	imgPath := utils.ImageUpload(file, unameCookie.Value)
	sidCookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Error(err)
		return api.Request{}, err
	}

	// create return data
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.Nickname] = nickname
	ret[api.ProfilePic] = imgPath
	ret[api.SessionId] = sidCookie.Value
	ret[api.Username] = unameCookie.Value
	req := api.Request{
		Id:   rid,
		Type: "EDIT",
		Data: ret,
	}
	return req, nil
}

func processEditRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.EDIT_SUCCESS:
		qs := utils.CreateQueryString("Edit Success!")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	case api.EDIT_FAILED:
		qs := utils.CreateQueryString("Edit Failed...")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	}
}

// **********************************
// *********** REGISTER *************
// **********************************
func (srv *HTTPServer) registerUser(w http.ResponseWriter, r *http.Request) {
	req := createRegisterReq(r)
	log.Info("Create register request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Register failed, please try again in a while")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}

	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Register failed, please try again in a while")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Register failed, please try again in a while")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive register response", res)

	// PROCESS RESPONSE
	processRegisterRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createRegisterReq(r *http.Request) api.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.Username] = username
	ret[api.PwHash] = security.Hash(password)
	ret[api.Nickname] = nickname
	req := api.Request{
		Id:   rid,
		Type: "REGISTER",
		Data: ret,
	}
	return req
}

func processRegisterRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.INSERT_SUCCESS:
		qs := utils.CreateQueryString("Account created!")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
	case api.INSERT_FAILED:
		qs := utils.CreateQueryString("Account creation failed, please try again!")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
	}
}

// ********************************
// *********** LOGOUT *************
// ********************************
func (srv *HTTPServer) logout(w http.ResponseWriter, r *http.Request) {
	req := createLogoutReq(r)
	log.Info("Create logout request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Logout failed, please try again in a while")
		http.Redirect(w, r, "/home"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Logout failed, please try again in a while")
		http.Redirect(w, r, "/home"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Logout failed, please try again in a while")
		http.Redirect(w, r, "/home"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive logout response", res)

	// PROCESS RESPONSE
	processLogoutRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createLogoutReq(r *http.Request) api.Request {
	c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.SessionId] = c.Value
	req := api.Request{
		Id:   rid,
		Type: "LOGOUT",
		Data: ret,
	}
	return req
}

func processLogoutRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.LOGOUT_SUCCESS:
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
func (srv *HTTPServer) home(w http.ResponseWriter, r *http.Request) {
	req := createHomeReq(r)
	log.Info("Create home request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Home failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Home failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Home failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive home response", res)

	// PROCESS RESPONSE
	processHomeRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

// retrieves the current user based on session cookie.
func createHomeReq(r *http.Request) api.Request {
	cookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	// get the user details of this session id
	rid := r.Header.Get(api.RequestIdHeader)
	data := make(map[string]string)
	data[api.SessionId] = cookie.Value
	req := api.Request{
		Id:   rid,
		Type: "HOME",
		Data: data,
	}
	return req
}

func processHomeRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.CREDENTIALS_VALID:
		renderTemplate(w, "home", res.Data)
	case api.CREDENTIALS_INVALID:
		qs := utils.CreateQueryString("User not found, please login!")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
	}
}

// ***************************************
// *********** HTTP HANDLERS *************
// ***************************************
func (srv *HTTPServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

func (srv *HTTPServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "login", desc)
	case http.MethodPost:
		srv.login(w, r)
	default:
		log.Fatalln("Unused method" + r.Method)
	}
}

func (srv *HTTPServer) editHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/login", http.StatusForbidden)
		return
	}
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "edit", desc)
	case http.MethodPost:
		srv.edit(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func (srv *HTTPServer) registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "register", desc)
	case http.MethodPost:
		srv.registerUser(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func (srv *HTTPServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	srv.home(w, r)
}

func (srv *HTTPServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	srv.logout(w, r)
}

func initLogger(logLevel string, logOutput string) {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "Jan _2 15:04:05.000000"
	customFormatter.FullTimestamp = true
	customFormatter.ForceColors = false
	customFormatter.DisableColors = true
	log.SetFormatter(customFormatter)
	err := os.Remove("http.txt")
	if err != nil {
		log.Error(err)
	}

	switch logLevel {
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		fallthrough
	default:
		log.SetLevel(log.InfoLevel)
	}

	switch logOutput {
	case "NONE":
		log.SetOutput(ioutil.Discard)
	case "FILE":
		file, err := os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Error(err)
		}
		log.SetOutput(file)
	case "ALL":
		file, err := os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Error(err)
		}
		log.SetOutput(io.MultiWriter(file, os.Stdout))
	case "STDERR":
		fallthrough
	default:
		// Stderr is set by default
	}
}

func initPool() pool.Pool {
	myPool := new(pool.TcpPool).NewTcpPool(pool.TcpPoolConfig{
		InitialSize: 1000,
		MaxSize:     1200,
		Factory: func() (net.Conn, error) {
			return net.Dial("tcp", "127.0.0.1:9090")
		},
	})
	return myPool
}

func initPoolSP() poolSP.Pool {
	//factory Specify the method to create the connection
	factory := func() (interface{}, error) { return net.Dial("tcp", "127.0.0.1:9090") }

	//close Specify the method to close the connection
	closee := func(v interface{}) error { return v.(net.Conn).Close() }

	//ping Specify the method to detect whether the connection is invalid
	ping := func(v interface{}) error { return nil }

	//Create a connection pool: Initialize the number of connections to 5, the maximum idle connection is 20, and the maximum concurrent connection is 30
	poolConfig := &poolSP.Config{
		InitialCap: 5,
		MaxIdle:    5,
		MaxCap:     5,
		Factory:    factory,
		Close:      closee,
		Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 60 * time.Second,
	}
	p, err := poolSP.NewChannelPool(poolConfig)
	if err != nil {
		panic(err)
	}
	return p
}

func (srv *HTTPServer) withRequestId(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(api.RequestIdHeader)
		if rid == "" {
			rid = uuid.NewV4().String()
			r.Header.Set(api.RequestIdHeader, rid)
		}
		handler(w, r)
	}
}

func (srv *HTTPServer) Start() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
	initLogger(*logLevel, *logOutput)

	log.Info("HTTP server listening on port ", srv.Port)

	// have the server listen on required routes
	http.HandleFunc("/", srv.withRequestId(srv.rootHandler))
	http.HandleFunc("/login", srv.withRequestId(srv.loginHandler))
	http.HandleFunc("/logout", srv.withRequestId(srv.logoutHandler))
	http.HandleFunc("/home", srv.withRequestId(srv.homeHandler))
	http.HandleFunc("/edit", srv.withRequestId(srv.editHandler))
	http.HandleFunc("/register", srv.withRequestId(srv.registerHandler))
	http.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir("./images"))))
	server := &http.Server{
		Addr:         ":" + srv.Port,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
	//log.Fatal(http.ListenAndServe(fmt.Sprintf("%v:%v", srv.Hostname, srv.Port), http.DefaultServeMux))
}

func (srv *HTTPServer) Stop() {
	srv.TcpPool.Stats()
	log.Info("HTTP server stopped.")
}

func main() {
	flag.Parse()
	log.Info("LOGLEVEL: " + *logLevel)
	log.Info("LOGOUTPUT: " + *logOutput)

	server := HTTPServer{
		Hostname: "127.0.0.1",
		Port:     "8080",
		TcpPool:  initPool(),
	}

	defer server.Stop()
	server.Start()
}
