package main

import (
	"encoding/json"
	"example.com/kendrick/auth"
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

func handleEditReq(req protocol.Request) {
	//data := req.Data
	// TODO
}

// Invokes the relevant request handler
func handleData(req protocol.Request, conn net.Conn) protocol.Response {
	switch req.Source {
	case "LOGIN":
		return handleLoginReq(req)
	case "EDIT":
		handleEditReq(req)
		_, err := conn.Write([]byte("Edit request\n"))
		if err != nil {
			log.Panicln(err)
		}
	default:
		log.Fatalln("Unknown request source " + req.Source)
	}
	return protocol.Response{}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	message := receiveData(conn)
	response := handleData(message, conn)
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
