package main

import (
	"encoding/json"
	"example.com/kendrick/auth"
	"example.com/kendrick/common"
	database "example.com/kendrick/mysql-db"
	"example.com/kendrick/protocol"
	"io"
	"log"
	"net"
)

// ********************************
// *********** COMMON *************
// ********************************
func handleConn(conn net.Conn) {
	defer conn.Close()
	message := receiveData(conn)
	response := handleData(message)
	sendResponse(response, conn)
}

// Invokes the relevant request handler
func handleData(req protocol.Request) protocol.Response {
	switch req.Source {
	case "LOGIN":
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
		log.Fatalln("Unknown request source " + req.Source)
	}
	return protocol.Response{}
}

func receiveData(conn net.Conn) protocol.Request {
	dec := json.NewDecoder(conn)
	var m protocol.Request
	if err := dec.Decode(&m); err == io.EOF {
		// do nothing
	} else if err != nil {
		log.Fatalln(err)
	}
	log.Println(m)
	return m
}

func sendResponse(data protocol.Response, conn net.Conn) {
	res := protocol.CreateResponse(data)
	_, err := conn.Write(res)
	common.Display("SENDING RESPONSE: ", string(res))
	if err != nil {
		log.Panicln(err)
	}
}

// Checks the validity of username and password hash in login request.
func handleLoginReq(req protocol.Request) protocol.Response {
	data := req.Data
	username := data[protocol.Username]
	pw := data[protocol.PwPlain]
	common.Display("HANDLING LOGIN REQ: ", data)

	if auth.IsValidPassword(username, pw) {
		sid := auth.CreateSession(username)
		data := make(map[string]string)
		data[protocol.Username] = username
		data[protocol.SessionId] = sid
		res := protocol.Response{
			Code:        protocol.USER_FOUND,
			Description: "Login for " + username + " succeeded",
			Data:        data,
		}
		common.Display("VALID PW, SENDING RESPONSE: ", res)
		return res
	}
	res := protocol.Response{
		Code:        protocol.NO_SUCH_USER,
		Description: "Login for " + username + " failed",
		Data:        nil,
	}
	common.Display("INVALID PW, SENDING RESPONSE: ", res)
	return res
}

func handleEditReq(req protocol.Request) protocol.Response {
	data := req.Data
	sid := data[protocol.SessionId]
	nickname := data[protocol.Nickname]
	picPath := data[protocol.ProfilePic]
	username := auth.GetSessionUser(sid)
	common.Display("HANDLING EDIT REQ: ", data)

	// Find the username, and replace the relevant details
	numRows := database.UpdateUser(username, nickname, picPath)
	if numRows == 1 {
		res := protocol.Response{
			Code:        protocol.EDIT_SUCCESS,
			Description: "Edited " + username + " successfully",
			Data:        nil,
		}
		common.Display("VALID EDIT, SENDING RESPONSE: ", res)
		return res
	}
	res := protocol.Response{
		Code:        protocol.EDIT_FAILED,
		Description: "Editing " + username + " failed",
		Data:        nil,
	}
	common.Display("INVALID EDIT, SENDING RESPONSE: ", res)
	return res
}

func handleLogoutReq(req protocol.Request) protocol.Response {
	data := req.Data
	sid := data[protocol.SessionId]
	common.Display("HANDLING LOGOUT REQ: ", data)

	username := auth.DelSessionUser(sid)
	res := protocol.Response{
		Code:        protocol.LOGOUT_SUCCESS,
		Description: "Logged out: " + sid + " " + username,
		Data:        nil,
	}
	common.Display("VALID LOGOUT, SENDING RESPONSE: ", res)
	return res
}

func handleRegReq(req protocol.Request) protocol.Response {
	data := req.Data
	nickname := data[protocol.Nickname]
	username := data[protocol.Username]
	password := data[protocol.PwHash]
	common.Display("HANDLING REGISTER REQ: ", data)

	numRows := database.InsertUser(username, password, nickname)
	if numRows == 1 {
		res := protocol.Response{
			Code:        protocol.INSERT_SUCCESS,
			Description: "INSERT: " + username + " " + password + " " + nickname,
			Data:        nil,
		}
		common.Display("VALID REGISTER, SENDING RESPONSE: ", res)
		return res
	}
	res := protocol.Response{
		Code:        protocol.INSERT_FAILED,
		Description: "INSERT failed: " + username + " " + password + " " + nickname,
		Data:        nil,
	}
	common.Display("INVALID REGISTER, SENDING RESPONSE: ", res)
	return res
}

func handleHomeReq(req protocol.Request) protocol.Response {
	data := req.Data
	sid := data[protocol.SessionId]
	common.Display("HANDLING HOME REQ: ", data)

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
			Code:        protocol.USER_FOUND,
			Description: "User " + username + " found!",
			Data:        ret,
		}
		common.Display("VALID HOME, SENDING RESPONSE: ", response)
		return response
	}
	response := protocol.Response{
		Code:        protocol.NO_SUCH_USER,
		Description: "User " + username + " not found...",
		Data:        nil,
	}
	common.Display("INVALID HOME, SENDING RESPONSE: ", response)
	return response
}

func init() {
	database.Connect()
	database.DeleteSessions()
}

func main() {
	common.Display("TCP Server listening on port 8081", nil)
	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalln(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		go handleConn(conn)
	}
}
