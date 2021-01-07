package main

import (
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/security"
	"example.com/kendrick/internal/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// **********************************
// *********** REGISTER *************
// **********************************
func (srv *HTTPServer) registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		desc := r.URL.Query().Get("desc")
		renderTemplate(w, "register", desc)
	case http.MethodPost:
		srv.registerUser(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func (srv *HTTPServer) registerUser(w http.ResponseWriter, r *http.Request) {
	req := createRegisterReq(r)
	log.Info("Create register request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Register failed, please try again in a while")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}

	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Register failed, please try again in a while")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Register failed, please try again in a while")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive register response", res)

	// PROCESS RESPONSE
	processRegisterRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createRegisterReq(r *http.Request) api.Request {
	username := r.FormValue("username")
	password := r.FormValue("password")
	nickname := r.FormValue("nickname")
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.Username] = username
	ret[api.PwHash] = security.Hash(password)
	ret[api.Nickname] = nickname
	req := api.Request{
		Id:   rid,
		Type: "REGISTER",
		Data: ret,
	}
	return req
}

func processRegisterRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.INSERT_SUCCESS:
		qs := utils.CreateQueryString("Account created!")
		http.Redirect(w, r, "/login"+qs, http.StatusSeeOther)
	case api.INSERT_FAILED:
		qs := utils.CreateQueryString("Account creation failed, please try again!")
		http.Redirect(w, r, "/register"+qs, http.StatusSeeOther)
	}
}
