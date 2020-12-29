package main

import "net/http"

func (srv *HTTPServer) homeHandler(w http.ResponseWriter, r *http.Request) {
	if user, ok := fromContext(r.Context()); ok {
		renderTemplate(w, "home", user)
		return
	}
	http.Redirect(w, r, "/login", http.StatusUnauthorized)
}
