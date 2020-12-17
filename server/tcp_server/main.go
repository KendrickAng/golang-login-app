package main

import (
	"encoding/gob"
	"example.com/kendrick/auth"
	database "example.com/kendrick/database"
	"example.com/kendrick/protocol"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"time"
)

type TCPServer struct {
	Port string
}

const LOG_LEVEL = log.DebugLevel

// ********************************
// *********** COMMON *************
// ********************************
func handleError(rid string, conn net.Conn, err error) {
	if err != nil {
		logger := log.WithFields(log.Fields{protocol.RequestId: rid})
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

func handleConn(conn net.Conn) {
	// TODO: don't close the conn, persist
	for {
		log.Error("Receive data")
		message, err := receiveData(conn)
		log.Error("You should not be here")
		if err != nil {
			log.Info("Receive request failed", message)
			handleError(message.Id, conn, err)
			return
		}
		log.Info("Receive request success", message)
		response := handleData(message)
		log.Info("Sending response", message)
		err = sendResponse(response, conn)
		if err != nil {
			log.Info("Send response failed", message)
			handleError(message.Id, conn, err)
			return
		}
		log.Info("Send response success", message)
	}
}

// Invokes the relevant request handler
func handleData(req protocol.Request) protocol.Response {
	switch req.Type {
	case "LOGIN":
		// profiling.RecordLogin("LOGIN\n")
		return handleLoginReq(req)
	case "EDIT":
		return handleEditReq(req)
	case "LOGOUT":
		return handleLogoutReq(req)
	case "REGISTER":
		return handleRegReq(req)
	case "HOME":
		return handleHomeReq(req)
	default:
		log.Error("Unknown request source " + req.Type)
	}
	return protocol.Response{}
}

func receiveData(conn net.Conn) (protocol.Request, error) {
	log.Debug("Receiving request")
	var m protocol.Request
	err := gob.NewDecoder(conn).Decode(&m)
	if err != nil {
		return protocol.Request{}, err
	}
	log.Debug("Received request", m)
	return m, nil
}

func sendResponse(data protocol.Response, conn net.Conn) error {
	log.Debug("Sending response", data)
	err := gob.NewEncoder(conn).Encode(data)
	if err != nil {
		return err
	}
	log.Debug("Sent response", data)
	return nil
}

// Checks the validity of username and password hash in login request.
func handleLoginReq(req protocol.Request) protocol.Response {
	data := req.Data
	username := data[protocol.Username]
	pw := data[protocol.PwPlain]
	log.WithFields(log.Fields{
		protocol.Username: username,
		protocol.PwPlain:  pw,
	}).Debug("Handling login request")

	if auth.IsValidPassword(username, pw) {
		sid := auth.CreateSession(username)
		data := make(map[string]string)
		data[protocol.Username] = username
		data[protocol.SessionId] = sid
		res := protocol.Response{
			Id:          req.Id,
			Code:        protocol.CREDENTIALS_INVALID,
			Description: "Login for " + username + " succeeded",
			Data:        data,
		}
		log.Debug("Valid password")
		return res
	}
	res := protocol.Response{
		Id:          req.Id,
		Code:        protocol.CREDENTIALS_VALID,
		Description: "Login for " + username + " failed",
		Data:        nil,
	}
	log.Debug("Invalid password")
	return res
}

func handleEditReq(req protocol.Request) protocol.Response {
	data := req.Data
	sid := data[protocol.SessionId]
	nickname := data[protocol.Nickname]
	picPath := data[protocol.ProfilePic]
	username := auth.GetSessionUser(sid)
	log.WithFields(log.Fields{
		protocol.RequestId:  req.Id,
		protocol.SessionId:  sid,
		protocol.Username:   username,
		protocol.Nickname:   nickname,
		protocol.ProfilePic: picPath,
	}).Debug("Handling edit request")

	// Find the username, and replace the relevant details
	numRows := database.UpdateUser(username, nickname, picPath)
	if numRows == 1 {
		res := protocol.Response{
			Id:          req.Id,
			Code:        protocol.EDIT_SUCCESS,
			Description: "Edited " + username + " successfully",
			Data:        nil,
		}
		log.Debug("Valid edit")
		return res
	}
	res := protocol.Response{
		Id:          req.Id,
		Code:        protocol.EDIT_FAILED,
		Description: "Editing " + username + " failed",
		Data:        nil,
	}
	log.Debug("Invalid edit")
	return res
}

func handleLogoutReq(req protocol.Request) protocol.Response {
	data := req.Data
	sid := data[protocol.SessionId]
	log.WithFields(log.Fields{
		protocol.RequestId: req.Id,
		protocol.SessionId: sid,
	}).Debug("Handling logout request")

	username := auth.DelSessionUser(sid)
	res := protocol.Response{
		Id:          req.Id,
		Code:        protocol.LOGOUT_SUCCESS,
		Description: "Logged out: " + sid + " " + username,
		Data:        nil,
	}
	log.Debug("Valid logout")
	return res
}

func handleRegReq(req protocol.Request) protocol.Response {
	data := req.Data
	nickname := data[protocol.Nickname]
	username := data[protocol.Username]
	password := data[protocol.PwHash]
	log.WithFields(log.Fields{
		protocol.RequestId: req.Id,
		protocol.Username:  username,
		protocol.PwHash:    password,
		protocol.Nickname:  nickname,
	}).Debug("Handling register request")

	numRows := database.InsertUser(username, password, nickname)
	if numRows == 1 {
		res := protocol.Response{
			Id:          req.Id,
			Code:        protocol.INSERT_SUCCESS,
			Description: "INSERT: " + username + " " + password + " " + nickname,
			Data:        nil,
		}
		log.Debug("Valid register")
		return res
	}
	res := protocol.Response{
		Id:          req.Id,
		Code:        protocol.INSERT_FAILED,
		Description: "INSERT failed: " + username + " " + password + " " + nickname,
		Data:        nil,
	}
	log.Debug("Invalid register")
	return res
}

func handleHomeReq(req protocol.Request) protocol.Response {
	data := req.Data
	sid := data[protocol.SessionId]
	log.WithFields(log.Fields{
		protocol.RequestId: req.Id,
		protocol.SessionId: sid,
	}).Debug("Handling home request")

	username := auth.GetSessionUser(sid)
	log.Println(sid, username)
	rows := database.GetUser(username)
	log.Println(rows)
	if len(rows) == 1 {
		ret := make(map[string]string)
		ret[protocol.Username] = rows[0].Username
		ret[protocol.Nickname] = rows[0].Nickname
		ret[protocol.ProfilePic] = rows[0].ProfilePic
		response := protocol.Response{
			Id:          req.Id,
			Code:        protocol.CREDENTIALS_INVALID,
			Description: "User " + username + " found!",
			Data:        ret,
		}
		log.Debug("Valid home request")
		return response
	}
	response := protocol.Response{
		Id:          req.Id,
		Code:        protocol.CREDENTIALS_VALID,
		Description: "User " + username + " not found...",
		Data:        nil,
	}
	log.Debug("Invalid home request")
	return response
}

func initLogger() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "Jan _2 15:04:05.000000"
	customFormatter.FullTimestamp = true
	customFormatter.ForceColors = false
	customFormatter.DisableColors = true
	log.SetFormatter(customFormatter)
	err := os.Remove("tcp.txt")
	if err != nil {
		log.Println(err)
	}
	_, err = os.OpenFile("tcp.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	//file, err := os.OpenFile("tcp.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Println(err)
	}
	//log.SetOutput(ioutil.Discard)
	//log.SetOutput(file)
	//log.SetOutput(io.MultiWriter(file, os.Stdout))
	log.SetLevel(LOG_LEVEL)
}

func (srv *TCPServer) Start() {
	database.Connect()
	database.DeleteSessions()
	initLogger()

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
		go handleConn(conn)
	}
}

func (srv *TCPServer) Stop() {
	database.Disconnect()
	log.Info("HTTP server stopped.")
}

func main() {
	server := TCPServer{Port: "9090"}
	defer server.Stop()
	server.Start()
}
