package main

import (
	"encoding/json"
	"example.com/kendrick/auth"
	"example.com/kendrick/mysql-db"
	"example.com/kendrick/protocol"
	"io"
	"log"
	"net"
)

func handleConn(conn net.Conn) {
	defer conn.Close()
	handleReq(conn)
}

func handleReq(conn net.Conn) {
	dec := json.NewDecoder(conn)
	var m protocol.Request
	if err := dec.Decode(&m); err == io.EOF {
		// do nothing
	} else if err != nil {
		log.Fatalln(err)
	}
	log.Println(m)
	mux(m, conn)
}

// Checks the validity of username and password hash in login request.
func handleLoginReq(req protocol.Request) protocol.Response {
	data := req.Data
	username := data[protocol.Username]
	pw := data[protocol.PwPlain]

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
func mux(req protocol.Request, conn net.Conn) {
	switch req.Source {
	case "LOGIN":
		res := protocol.CreateResponse(handleLoginReq(req))
		_, err := conn.Write(res)

		log.Println("Handling LOGIN request: " + string(res))
		if err != nil {
			log.Panicln(err)
		}
	case "EDIT":
		handleEditReq(req)
		_, err := conn.Write([]byte("Edit request\n"))
		if err != nil {
			log.Panicln(err)
		}
	}
}

func init() {
	database.Connect()
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
