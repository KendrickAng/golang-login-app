package main

import (
	"encoding/gob"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/auth"
	database "example.com/kendrick/internal/tcp_server/database"
	"example.com/kendrick/internal/tcp_server/session"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

type TCPServer struct {
	Port    string
	DB      database.DB
	SessMgr session.SessionManager
}

var (
	logOutput = flag.String(
		"logOutput",
		"",
		"Logrus log output, NONE/FILE/STDERR/ALL, default: STDERR",
	)
	logLevel   = flag.String("logLevel", "", "Logrus log level, DEBUG/ERROR/INFO, default: INFO")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

// ********************************
// *********** COMMON *************
// ********************************
func handleError(rid string, conn net.Conn, err error) {
	if err != nil {
		logger := log.WithFields(log.Fields{api.RequestId: rid})
		if err == io.EOF {
			// connection closed by HTTP server
			logger.Error(err, " closing connection")
			conn.Close()
		} else if e, ok := err.(*net.OpError); ok {
			if e.Timeout() {
				// If SetDeadline triggers
				logger.Error("Timeout (likely from SetDeadline) ", e)
			} else {
				logger.Error("OpError: ", e)
			}
		} else {
			logger.Error(err)
		}
	}
}

func (srv *TCPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	var err error
	for err != io.EOF {
		msgs := api.Request{}
		err = decoder.Decode(&msgs)
		if err != nil {
			if err != io.EOF {
				log.Error(err) // e.g extra data in buffer
			}
			continue
		}
		log.Info("Receive request success", msgs)
		response := srv.handleData(&msgs)
		log.Info("Sending response", response)
		err = encoder.Encode(response)
		if err != nil {
			handleError(msgs.Id, conn, err)
			return
		}
		log.Info("Send response success", msgs)
	}
}

// Invokes the relevant request handler
func (srv *TCPServer) handleData(req *api.Request) api.Response {
	switch req.Type {
	case "LOGIN":
		return srv.handleLoginReq(req)
	case "EDIT":
		return srv.handleEditReq(req)
	case "LOGOUT":
		return srv.handleLogoutReq(req)
	case "REGISTER":
		return srv.handleRegReq(req)
	case "HOME":
		return srv.handleHomeReq(req)
	case "GET_SESSION":
		return srv.handleSessReq(req)
	default:
		log.Error("Unknown request source " + req.Type)
	}
	return api.Response{}
}

func (srv *TCPServer) handleSessReq(req *api.Request) api.Response {
	sid := req.Data[api.SessionId]
	log.WithFields(log.Fields{
		api.SessionId: sid,
		api.RequestId: req.Id,
	}).Debug("Handling session request")

	sess, err := srv.SessMgr.GetSession(sid)
	if err != nil {
		log.Error(err)
		return api.Response{
			Id:          req.Id,
			Code:        api.GET_SESS_FAILED,
			Description: err.Error(),
			Data:        nil,
		}
	}
	ret := make(map[string]string)
	ret[api.Username] = sess.GetUsername()
	ret[api.Nickname] = sess.GetNickname()
	ret[api.PwHash] = sess.GetPwHash()
	ret[api.ProfilePic] = sess.GetProfilePic()
	return api.Response{
		Id:          req.Id,
		Code:        api.GET_SESS_SUCCESS,
		Description: "Success",
		Data:        ret,
	}
}

// Checks the validity of username and password hash in login request.
func (srv *TCPServer) handleLoginReq(req *api.Request) api.Response {
	data := req.Data
	username := data[api.Username]
	pw := data[api.PwPlain]
	log.WithFields(log.Fields{
		api.Username: username,
		api.PwPlain:  pw,
	}).Debug("Handling login request")

	user, err := srv.DB.GetUser(username)
	if err != nil {
		log.Debug("Invalid password")
		return api.Response{
			Id:          req.Id,
			Code:        api.LOGIN_FAILED,
			Description: err.Error(),
			Data:        nil,
		}
	}

	if auth.IsValidPassword(user, pw) {
		sess, err := srv.SessMgr.CreateSession(user)
		if err != nil {
			log.Error(err)
			return api.Response{
				Id:          req.Id,
				Code:        api.LOGIN_FAILED,
				Description: err.Error(),
				Data:        nil,
			}
		}
		ret := make(map[string]string)
		ret[api.Username] = username
		ret[api.SessionId] = sess.GetSessID()
		res := api.Response{
			Id:          req.Id,
			Code:        api.LOGIN_SUCCESS,
			Description: "Login for " + username + " succeeded",
			Data:        ret,
		}
		log.Debug("Valid password")
		return res
	}
	res := api.Response{
		Id:          req.Id,
		Code:        api.LOGIN_FAILED,
		Description: "Login for " + username + " failed",
		Data:        nil,
	}
	log.Debug("Invalid password")
	return res
}

func (srv *TCPServer) handleEditReq(req *api.Request) api.Response {
	data := req.Data
	sid := data[api.SessionId]
	nickname := data[api.Nickname]
	picPath := data[api.ProfilePic]
	username := data[api.Username]
	log.WithFields(log.Fields{
		api.RequestId:  req.Id,
		api.SessionId:  sid,
		api.Username:   username,
		api.Nickname:   nickname,
		api.ProfilePic: picPath,
	}).Debug("Handling edit request")

	// Find the username, and replace the relevant details
	numRows := srv.DB.UpdateUser(username, nickname, picPath)
	if numRows == 1 {
		res := api.Response{
			Id:          req.Id,
			Code:        api.EDIT_SUCCESS,
			Description: "Edited " + username + " successfully",
			Data:        nil,
		}
		log.Debug("Valid edit")
		return res
	}
	res := api.Response{
		Id:          req.Id,
		Code:        api.EDIT_FAILED,
		Description: "Editing " + username + " failed",
		Data:        nil,
	}
	log.Debug("Invalid edit")
	return res
}

func (srv *TCPServer) handleLogoutReq(req *api.Request) api.Response {
	data := req.Data
	sid := data[api.SessionId]
	log.WithFields(log.Fields{
		api.RequestId: req.Id,
		api.SessionId: sid,
	}).Debug("Handling logout request")

	err := srv.SessMgr.DeleteSession(sid)
	if err != nil {
		return api.Response{
			Id:          req.Id,
			Code:        api.LOGIN_FAILED,
			Description: err.Error(),
			Data:        nil,
		}
	}
	res := api.Response{
		Id:          req.Id,
		Code:        api.LOGOUT_SUCCESS,
		Description: "Logged out session: " + sid,
		Data:        nil,
	}
	log.Debug("Valid logout")
	return res
}

func (srv *TCPServer) handleRegReq(req *api.Request) api.Response {
	data := req.Data
	nickname := data[api.Nickname]
	username := data[api.Username]
	password := data[api.PwHash]
	log.WithFields(log.Fields{
		api.RequestId: req.Id,
		api.Username:  username,
		api.PwHash:    password,
		api.Nickname:  nickname,
	}).Debug("Handling register request")

	numRows := srv.DB.InsertUser(username, password, nickname)
	if numRows == 1 {
		res := api.Response{
			Id:          req.Id,
			Code:        api.INSERT_SUCCESS,
			Description: "INSERT: " + username + " " + password + " " + nickname,
			Data:        nil,
		}
		log.Debug("Valid register")
		return res
	}
	res := api.Response{
		Id:          req.Id,
		Code:        api.INSERT_FAILED,
		Description: "INSERT failed: " + username + " " + password + " " + nickname,
		Data:        nil,
	}
	log.Debug("Invalid register")
	return res
}

func (srv *TCPServer) handleHomeReq(req *api.Request) api.Response {
	data := req.Data
	sid := data[api.SessionId]
	log.WithFields(log.Fields{
		api.RequestId: req.Id,
		api.SessionId: sid,
	}).Debug("Handling home request")

	sess, err := srv.SessMgr.GetSession(sid)
	if err != nil {
		log.Error(err)
		return api.Response{
			Id:          req.Id,
			Code:        api.HOME_FAILED,
			Description: err.Error(),
			Data:        nil,
		}
	}
	username := sess.GetUsername()
	user, err := srv.DB.GetUser(username)
	if err == nil {
		ret := make(map[string]string)
		ret[api.Username] = user.Username
		ret[api.Nickname] = user.Nickname
		ret[api.ProfilePic] = user.ProfilePic
		response := api.Response{
			Id:          req.Id,
			Code:        api.HOME_SUCCESS,
			Description: "User " + username + " found!",
			Data:        ret,
		}
		log.Debug("Valid home request")
		return response
	}
	response := api.Response{
		Id:          req.Id,
		Code:        api.HOME_FAILED,
		Description: err.Error(),
		Data:        nil,
	}
	log.Debug("Invalid home request")
	return response
}

func initLogger(logLevel string, logOutput string) {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "Jan _2 15:04:05.000000"
	customFormatter.FullTimestamp = true
	customFormatter.ForceColors = false
	customFormatter.DisableColors = true
	log.SetFormatter(customFormatter)
	err := os.Remove("tcp.txt")
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
		file, err := os.OpenFile("tcp.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Error(err)
		}
		log.SetOutput(file)
	case "ALL":
		file, err := os.OpenFile("tcp.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
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

func (srv *TCPServer) Start() {
	initLogger(*logLevel, *logOutput)

	log.Info("TCP Server listening on port ", srv.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", srv.Port))
	if err != nil {
		log.Panicln(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panicln(err)
		}

		if c, ok := conn.(*net.TCPConn); ok {
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(time.Second * 60)
		}
		go srv.handleConn(conn)
	}
}

func (srv *TCPServer) Stop() {
	srv.SessMgr.Stop()
	log.Info("HTTP server stopped.")
}

func main() {
	// allow server to release resources when done
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// cpu profiling
	flag.Parse()
	log.Info("CPUPROFILE: " + *cpuprofile)
	log.Info("LOGLEVEL: " + *logLevel)
	log.Info("LOGOUTPUT: " + *logOutput)
	if *cpuprofile != "" {
		err := os.Remove(*cpuprofile)
		if err != nil {
			log.Error(err)
		}
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// session manager
	sessMgr, err := session.NewManager(4)
	if err != nil {
		log.Panicln(err)
	}
	// database for users
	db, err := database.NewDB()
	if err != nil {
		log.Panicln(err)
	}

	server := TCPServer{
		Port:    "9090",
		SessMgr: sessMgr,
		DB:      db,
	}
	defer server.Stop()
	go server.Start()

	<-done
	fmt.Println("SERVER STOPPED")
}
