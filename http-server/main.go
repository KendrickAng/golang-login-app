package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"example.com/kendrick/auth"
	"example.com/kendrick/http-server/fileio"
	"example.com/kendrick/protocol"
	"fmt"
	"html/template"
	"io"
	"log"
	_ "mime/multipart"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	TIMEOUT = time.Second * 5
)

var templates *template.Template

func renderTemplate(w http.ResponseWriter, tmpl string) {
	file := fmt.Sprintf("%s.html", tmpl)
	err := templates.ExecuteTemplate(w, file, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

// Main handler called when logging in
func loginReqHandler(w http.ResponseWriter, r *http.Request) {
	if auth.IsLoggedIn(r) {
		http.Redirect(w, r, "/edit", http.StatusSeeOther)
		return
	}

	// extract sent form data
	data := processLoginForm(r)

	// send the information to TCP server
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	request := protocol.CreateRequest(data)
	log.Println("Sending request: " + string(request))
	_, err = conn.Write(request)
	//_, err = fmt.Fprintf(conn, request)
	if err != nil {
		log.Panicln(err)
	}

	// receive the response with timeout
	err = conn.SetDeadline(time.Now().Add(TIMEOUT))
	if err != nil {
		log.Panicln(err)
	}
	dec := json.NewDecoder(conn)
	err = conn.SetDeadline(time.Time{})
	var res protocol.Response
	err = dec.Decode(&res)
	if err == io.EOF {
		log.Println("EOF!!!")
	} else if errors.Is(err, os.ErrDeadlineExceeded) {
		http.Error(w, "TCP Server timeout", http.StatusInternalServerError)
		return
	} else if err != nil {
		log.Fatalln(err)
	} else {
		log.Print("TCP Server response: ")
		log.Println(res)
		loginResHandler(w, r, res)
	}
}

// gets username and password from sent form
func processLoginForm(r *http.Request) protocol.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	ret := make(map[string]string)
	ret[protocol.Username] = username
	ret[protocol.PwPlain] = password
	return protocol.Request{
		Method: "POST",
		Source: "LOGIN",
		Data:   ret,
	}
}

func loginResHandler(w http.ResponseWriter, r *http.Request, res protocol.Response) {
	if res.Code != protocol.USER_FOUND {
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	username := res.Data[protocol.Username]
	sid := auth.CreateSession(username)
	http.SetCookie(w, &http.Cookie{
		Name:  auth.SESS_COOKIE_NAME,
		Value: sid,
	})
	log.Println("Created session: " + username + " " + sid)
}

// gets the submitted image (if any) and saves it, returning stored address
func processEditForm(r *http.Request) protocol.Request {
	// max size 1MB
	r.ParseMultipartForm(1 << 20)

	nickname := r.FormValue("nickname")
	file, _, err := r.FormFile("pic")
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()
	// TODO: Implement sessions
	imgPath, err := fileio.ImageUpload(file, "TEMPORARY HASH PLEASE REMOVE")
	if err != nil {
		log.Fatalln(err)
	}

	ret := make(map[string]string)
	ret[protocol.Nickname] = nickname
	ret[protocol.ProfilePic] = imgPath
	return protocol.Request{
		Method: "POST",
		Source: "EDIT",
		Data:   ret,
	}
}

// serves loginReqHandler form for users/queries TCP server
func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// TODO: get the authenticated user, 404 if no such user.
		// TODO: How to decide whether user is authenticated? cookies?
		renderTemplate(w, "login")
	case http.MethodPost:
		loginReqHandler(w, r)
	default:
		log.Fatalln("Unused method" + r.Method)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	//name := r.URL.Path[len("/edit/"):]
	// TODO: Check for authentication, redirect to loginReqHandler otherwise (how to authenticate?)
	// TODO: if logged in, serve the edit page
	switch r.Method {
	case http.MethodGet:
		renderTemplate(w, "edit")
	case http.MethodPost:
		// send the information to the TCP server
		data := processEditForm(r)
		conn, err := net.Dial("tcp", "localhost:8081")
		if err != nil {
			log.Fatalln(err)
		}
		defer conn.Close()
		fmt.Fprintf(conn, string(protocol.CreateRequest(data)))
		message, _ := bufio.NewReader(conn).ReadString('\n')
		log.Println("Server responds with " + message)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		renderTemplate(w, "register")
	//case http.MethodPost:
	//	createUser(r)
	//}
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func init() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
}

func main() {
	// have the server listen on required routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/edit", editHandler)
	http.HandleFunc("/register", registerHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
