package main

import "net/http"

func (srv *HTTPServer) rootHandler(w http.ResponseWriter, r *http.Request) {
	if isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}
