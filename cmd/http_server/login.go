package main

import (
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/auth"
	"example.com/kendrick/internal/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// *******************************
// *********** LOGIN *************
// *******************************
func (srv *HTTPServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "login", desc)
	case http.MethodPost:
		srv.login(w, r)
	default:
		log.Fatalln("Unused method" + r.Method)
	}
}

// Main handler called when logging in
func (srv *HTTPServer) login(w http.ResponseWriter, r *http.Request) {
	req := createLoginReq(r)
	log.Info("Create login request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Login failed, please try again in a while")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive login response", res)

	// PROCESS RESPONSE
	processLoginRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createLoginReq(r *http.Request) api.Request {
	log.Debug("Creating login request")
	username := r.FormValue("username")
	password := r.FormValue("password")
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.Username] = username
	ret[api.PwPlain] = password
	req := api.Request{
		Id:   rid,
		Type: "LOGIN",
		Data: ret,
	}
	log.WithFields(log.Fields{
		api.RequestId: rid,
		api.Username:  username,
		api.PwPlain:   password,
	}).Debug("Created login request")
	return req
}

func processLoginRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	logger := log.WithFields(log.Fields{
		api.RequestId: res.Id,
		api.ResCode:   res.Code,
		api.ResDesc:   res.Description,
	})
	logger.Debug("Processing login response")
	if res.Code == api.LOGIN_FAILED {
		qs := utils.CreateQueryString("No such account, please register first!")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	sid := res.Data[api.SessionId]
	username := res.Data[api.Username]
	http.SetCookie(w, &http.Cookie{
		Name:    auth.SESS_COOKIE_NAME,
		Value:   sid,
		Expires: time.Now().Add(COOKIE_TIMEOUT),
	})
	http.SetCookie(w, &http.Cookie{
		Name:    auth.USERNAME_COOKIE_NAME,
		Value:   username,
		Expires: time.Now().Add(COOKIE_TIMEOUT),
	})
	http.Redirect(w, r, "/home", http.StatusSeeOther)
	logger.Debug("Processed login response")
	return
}
