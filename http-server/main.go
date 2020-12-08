package main

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
)

var templates *template.Template

// TODO: Include later
type User struct {
	Nickname string
}

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

// serves login form for users/queries TCP server
func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// TODO: get the authenticated user, 404 if no such user.
		renderTemplate(w, "login")
	case "POST":
		// send the information to the TCP server
		username := r.FormValue("username")
		password := r.FormValue("password")
		log.Println("Login POST called " + username + " " + password)
		conn, err := net.Dial("tcp", "localhost:8081")
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Fprintf(conn, "This is a test: "+username+password)
	default:
		log.Fatalln("Unused method" + r.Method)
	}
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	//name := r.URL.Path[len("/edit/"):]
	// TODO: Check for authentication, redirect to login otherwise (how to authenticate?)
	// TODO: if logged in, serve the edit page
	switch r.Method {
	case "GET":
	case "POST":
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
	log.Fatal(http.ListenAndServe(":8080", nil))
}
