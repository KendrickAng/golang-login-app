package main

import (
	"encoding/json"
	"example.com/kendrick/auth"
	database "example.com/kendrick/mysql-db"
	"example.com/kendrick/protocol"
	"io"
	"log"
	"net"
)

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
	log.Println("Handling LOGIN request: " + string(res))
	if err != nil {
		log.Panicln(err)
	}
}

// Checks the validity of username and password hash in login request.
func handleLoginReq(req protocol.Request) protocol.Response {
	data := req.Data
	username := data[protocol.Username]
	pw := data[protocol.PwPlain]
	log.Println(username, pw)

	if auth.IsValidPassword(username, pw) {
		log.Println("Valid password")
		res := make(map[string]string)
		res[protocol.Username] = username
		return protocol.Response{
			Code:        protocol.USER_FOUND,
			Description: "Login for " + username + " succeeded",
			Data:        res,
		}
	}
	log.Println("Invalid password")
	return protocol.Response{
		Code:        protocol.NO_SUCH_USER,
		Description: "Login for " + username + " failed",
		Data:        nil,
	}
}

func handleEditReq(req protocol.Request) protocol.Response {
	data := req.Data
	username := data[protocol.Username]
	nickname := data[protocol.Nickname]
	picPath := data[protocol.ProfilePic]
	log.Println("Handling Edit Req: ", username, nickname, picPath)

	// Find the username, and replace the relevant details
	// TODO
	numRows := database.EditUser(username, nickname, picPath)
	if numRows == 1 {
		return protocol.Response{
			Code:        protocol.EDIT_SUCCESS,
			Description: "Edited " + username + " successfully",
			Data:        nil,
		}
	}
	return protocol.Response{
		Code:        protocol.EDIT_FAILED,
		Description: "Editing " + username + " failed",
		Data:        nil,
	}
}

// Invokes the relevant request handler
func handleData(req protocol.Request) protocol.Response {
	switch req.Source {
	case "LOGIN":
		return handleLoginReq(req)
	case "EDIT":
		return handleEditReq(req)
	case "LOGOUT":

	default:
		log.Fatalln("Unknown request source " + req.Source)
	}
	return protocol.Response{}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	message := receiveData(conn)
	response := handleData(message)
	sendResponse(response, conn)
}

func main() {
	log.Println("TCP Server listening on port 8081")
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
