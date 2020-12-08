package main

import (
	"bufio"
	"log"
	"net"
)

func handleErr(err error) {
	log.Fatalln(err)
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	handleReq(conn)
}

func handleReq(conn net.Conn) {
	// i := 0
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		ln := scanner.Text()
		log.Println(ln)
		if ln == "" { // End of headers
			break
		}
	}
}

func main() {
	log.Println("TCP Server listening on port 8081")
	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		handleErr(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			handleErr(err)
		}
		go handleConn(conn)
	}
}
