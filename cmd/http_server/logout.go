package main

import (
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/auth"
	"example.com/kendrick/internal/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// ********************************
// *********** LOGOUT *************
// ********************************
func (srv *HTTPServer) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(getSid(r)) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	srv.logout(w, r)
}

func (srv *HTTPServer) logout(w http.ResponseWriter, r *http.Request) {
	req := createLogoutReq(r)
	log.Info("Create logout request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Logout failed, please try again in a while")
		http.Redirect(w, r, "/home"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Logout failed, please try again in a while")
		http.Redirect(w, r, "/home"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Logout failed, please try again in a while")
		http.Redirect(w, r, "/home"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive logout response", res)

	// PROCESS RESPONSE
	processLogoutRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createLogoutReq(r *http.Request) api.Request {
	c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.SessionId] = c.Value
	req := api.Request{
		Id:   rid,
		Type: "LOGOUT",
		Data: ret,
	}
	return req
}

func processLogoutRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.LOGOUT_SUCCESS:
		// delete cookie
		c, _ := r.Cookie(auth.SESS_COOKIE_NAME)
		c = &http.Cookie{
			Name:   auth.SESS_COOKIE_NAME,
			Value:  "",
			MaxAge: -1,
		}
		http.SetCookie(w, c)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
