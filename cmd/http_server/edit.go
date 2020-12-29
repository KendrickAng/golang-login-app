package main

import (
	"errors"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/tcp_server/auth"
	"example.com/kendrick/internal/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

// ******************************
// *********** EDIT *************
// ******************************
func (srv *HTTPServer) editHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// ensure logged in
		if _, ok := fromContext(r.Context()); ok {
			desc := r.URL.Query().Get("desc")
			renderTemplate(w, "edit", desc)
			return
		}
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
	case http.MethodPost:
		srv.edit(w, r)
	default:
		log.Fatalln("Unused method " + r.Method)
	}
}

func (srv *HTTPServer) edit(w http.ResponseWriter, r *http.Request) {
	req, err := createEditReq(r)
	if err != nil {
		log.Println(err)
		http.Redirect(w, r, "/edit"+utils.CreateQueryString(err.Error()), http.StatusSeeOther)
		return
	}
	log.Info("Create edit request", req)
	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Debug("In getTcpConnPooled")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Edit failed, please try again later")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
		return
	}
	defer srv.TcpPool.Put(&conn)

	// SEND REQUEST
	log.Debug("Sending request", req)
	err = conn.Enc.Encode(req)
	if err != nil {
		log.Debug("In request")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Edit failed, please try again in a while")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
		return
	}
	log.Debug("Request sent", req)

	// RECEIVE RESPONSE
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		log.Info("In response")
		srv.handleError(req.Id, &conn, err)
		qs := utils.CreateQueryString("Edit failed, please try again in a while")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
		return
	}
	log.Info("Receive edit response", res)

	// PROCESS RESPONSE
	processEditRes(w, r, res)
	log.WithField(api.RequestId, req.Id).Info("Connection closed")
}

func createEditReq(r *http.Request) (api.Request, error) {
	// retrieve form values
	nickname := r.FormValue("nickname")
	file, header, err := r.FormFile("pic")
	if err != nil {
		log.Error(err)
	}
	// enforce max size
	if header.Size > IMG_MAXSIZE {
		err := errors.New("Image too large: maximum " + strconv.Itoa(IMG_MAXSIZE) + " bytes.")
		return api.Request{}, err
	}
	defer file.Close()

	// store image persistently
	user, ok := fromContext(r.Context())
	if !ok {
		return api.Request{}, errors.New("CreateEditReq: No username")
	}
	imgPath := utils.ImageUpload(file, user.Username)
	sidCookie, err := r.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil {
		log.Error(err)
		return api.Request{}, err
	}

	// create return data
	rid := r.Header.Get(api.RequestIdHeader)
	ret := make(map[string]string)
	ret[api.Nickname] = nickname
	ret[api.ProfilePic] = imgPath
	ret[api.SessionId] = sidCookie.Value
	ret[api.Username] = user.Username
	req := api.Request{
		Id:   rid,
		Type: "EDIT",
		Data: ret,
	}
	return req, nil
}

func processEditRes(w http.ResponseWriter, r *http.Request, res api.Response) {
	switch res.Code {
	case api.EDIT_SUCCESS:
		qs := utils.CreateQueryString("Edit Success!")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	case api.EDIT_FAILED:
		qs := utils.CreateQueryString("Edit Failed...")
		http.Redirect(w, r, "/edit"+qs, http.StatusSeeOther)
	}
}
