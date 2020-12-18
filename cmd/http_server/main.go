package main

import (
	"errors"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/http_server/pool"
	"example.com/kendrick/internal/tcp_server/auth"
	"example.com/kendrick/internal/tcp_server/security"
	"example.com/kendrick/internal/utils"
	"fmt"
	"github.com/satori/uuid"
	poolSP "github.com/silenceper/pool"
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
	IMG_MAXSIZE = 1 << 12 // 2^12
	LOG_LEVEL   = log.ErrorLevel
)

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
		log.Debug("In getTcpConn")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)
	//defer conn.Close()
	//err = sendReq(conn, req)

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
	//res, err := receiveRes(conn)
	//log.Debug("Receiving response")
	var res api.Response
	// TODO: FOUND THE SLOWDOWN AREA!
	err = conn.Dec.Decode(&res)
	//log.Debug("Received response", res)
	//if err != nil {
	//	log.Info("In response")
	//	srv.handleError(req.Id, &conn, err)
	//	qs := common.CreateQueryString("Login failed, please try again in a while")
	//	http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
	//	return
	//}
	//log.Info("Receive login response", res)
	//
	//// PROCESS RESPONSE
	//processLoginRes(w, r, res)
	//log.WithField(protocol.RequestId, req.Id).Info("Connection closed")
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
	http.SetCookie(w, &http.Cookie{
		Name:  auth.SESS_COOKIE_NAME,
		Value: sid,
	})
	http.Redirect(w, r, "/home", http.StatusSeeOther)
	logger.Debug("Processed login response")
	return
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
	cookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	imgPath := utils.ImageUpload(file, auth.GetSessionUser(cookie.Value))

	// create return data
	ret := make(map[string]string)
	ret[api.Nickname] = nickname
	ret[api.ProfilePic] = imgPath
	ret[api.SessionId] = cookie.Value
	req := api.Request{
		Type: "EDIT",
		Data: ret,
	}
	log.Debug("CREATED EDIT REQUEST: ", req)
	return req, nil
}

func processEditRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	utils.Print("PROCESSING EDIT RES: ", res)
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
//func registerUser(w http.ResponseWriter, r *http.Request) {
//	req := createRegisterReq(r)
//	conn := getTcpConn()
//	defer conn.Close()
//	// TODO
//	_ = sendReq(conn, req)
//	res, _ := receiveRes(conn)
//	processRegisterRes(w, r, res)
//}

func createRegisterReq(r *http.Request) api.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	ret := make(map[string]string)
	ret[api.Username] = username
	ret[api.PwHash] = security.Hash(password)
	ret[api.Nickname] = nickname
	req := api.Request{
		Type: "REGISTER",
		Data: ret,
	}
	utils.Print("CREATED REGISTER REQ: ", req)
	return req
}

func processRegisterRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	utils.Print("PROCESSING REGISTER RESPONSE: ", res)
	switch res.Code {
	case api.INSERT_SUCCESS:
		qs := utils.CreateQueryString("Account created!")
		http.Redirect(w, r, "/login?"+qs, http.StatusSeeOther)
	case api.INSERT_FAILED:
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

func createLogoutReq(r *http.Request) api.Request {
	c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
	ret := make(map[string]string)
	ret[api.SessionId] = c.Value
	req := api.Request{
		Type: "LOGOUT",
		Data: ret,
	}
	utils.Print("CREATED LOGOUT REQUEST: ", req)
	return req
}

func processLogoutRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	utils.Print("PROCESSING LOGOUT RESPONSE: ", res)
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
func createHomeReq(r *http.Request) api.Request {
	cookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Fatalln(err)
	}
	// get the user details of this session id
	data := make(map[string]string)
	data[api.SessionId] = cookie.Value
	req := api.Request{
		Type: "HOME",
		Data: data,
	}
	utils.Print("CREATED HOME REQUEST: ", req)
	return req
}

func processHomeRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	utils.Print("PROCESSING HOME RESPONSE: ", res)
	switch res.Code {
	case api.CREDENTIALS_INVALID:
		renderTemplate(w, "home", res.Data)
	case api.CREDENTIALS_VALID:
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
		//edit(w, r)
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
		//registerUser(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func (srv *HTTPServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	//home(w, r)
}

func (srv *HTTPServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(getSid(r)) {
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
		log.Error(err)
	}

	//_, err = os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	//file, err := os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Error(err)
	}
	//log.SetOutput(ioutil.Discard)
	//log.SetOutput(file)
	//log.SetOutput(io.MultiWriter(file, os.Stdout))
	log.SetLevel(LOG_LEVEL)
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
	//database.Connect()
	//profiling.InitLogFiles()
	initLogger()

	log.Info("HTTP server listening on port ", srv.Port)

	// have the server listen on required routes
	http.HandleFunc("/", srv.rootHandler)
	http.HandleFunc("/login", srv.withRequestId(srv.loginHandler))
	http.HandleFunc("/logout", srv.logoutHandler)
	http.HandleFunc("/home", srv.homeHandler)
	http.HandleFunc("/edit", srv.editHandler)
	http.HandleFunc("/register", srv.registerHandler)
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
	server := HTTPServer{
		Hostname: "127.0.0.1",
		Port:     "8080",
		TcpPool:  initPool(),
	}

	defer server.Stop()
	server.Start()
}
