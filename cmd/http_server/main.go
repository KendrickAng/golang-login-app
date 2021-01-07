package main

import (
	"context"
	"errors"
	"example.com/kendrick/api"
	"example.com/kendrick/internal/http_server/pool"
	"example.com/kendrick/internal/tcp_server/auth"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/satori/uuid"
	log "github.com/sirupsen/logrus"
	"html/template"
	"io"
	"io/ioutil"
	_ "mime/multipart"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
)

var (
	logOutput = flag.String(
		"logOutput",
		"",
		"Logrus log output, NONE/FILE/STDERR/ALL, default: STDERR",
	)
	logLevel    = flag.String("logLevel", "", "Logrus log level, DEBUG/ERROR/INFO, default: INFO")
	CONTEXT_KEY = uuid.NewV4()
)

const (
	COOKIE_TIMEOUT = time.Hour * 24
	IMG_MAXSIZE    = 1 << 12 // 2^12
)

type HTTPServer struct {
	Server   http.Server
	TcpPool  pool.Pool
	Hostname string
	Port     string
}

var templates *template.Template

// ********************************
// *********** COMMON *************
// ********************************
func isLoggedIn(sid string) bool {
	return sid != ""
}

// Gets the value of the session cookie. Returns "" if not present.
func getSid(req *http.Request) string {
	cookie, err := req.Cookie(auth.SESS_COOKIE_NAME)
	if err != nil && errors.Is(err, http.ErrNoCookie) {
		return ""
	}
	return cookie.Value
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	file := fmt.Sprintf("%s.html", tmpl)
	err := templates.ExecuteTemplate(w, file, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (srv *HTTPServer) handleError(rid string, conn *pool.TcpConn, err error) {
	if err != nil {
		logger := log.WithFields(log.Fields{api.RequestId: rid})
		if err == io.EOF {
			// connection closed by TCP server - tell the pool to destroy connection
			logger.Error(err)
			err := srv.TcpPool.Destroy(conn)
			if err != nil {
				logger.Error(err)
			}
		} else if errors.Is(err, os.ErrDeadlineExceeded) {
			// read deadline exceeded, do nothing
			logger.Error(err)
		} else {
			logger.Error("Others: ", err)
		}
	}
}

func (srv *HTTPServer) getTcpConnPooled() (pool.TcpConn, error) {
	conn, err := srv.TcpPool.Get()
	if err != nil {
		return pool.TcpConn{}, err
	}
	return conn, nil
}

func (srv *HTTPServer) getTcpConn() (net.Conn, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		return nil, err
	}
	return conn, err
}

func (srv *HTTPServer) getSession(sid string, rid string) (*api.User, error) {
	// construct request
	data := make(map[string]string)
	data[api.SessionId] = sid
	req := api.Request{
		Id:   rid,
		Type: "GET_SESSION",
		Data: data,
	}

	conn, err := srv.getTcpConnPooled()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer srv.TcpPool.Put(&conn)

	// send request
	err = conn.Enc.Encode(req)
	if err != nil {
		srv.handleError(req.Id, &conn, err)
		return nil, err
	}

	// receive response
	var res api.Response
	err = conn.Dec.Decode(&res)
	if err != nil {
		srv.handleError(req.Id, &conn, err)
		return nil, err
	}

	// process response
	return &api.User{
		Username:   res.Data[api.Username],
		Nickname:   res.Data[api.Nickname],
		PwHash:     res.Data[api.PwHash],
		ProfilePic: res.Data[api.ProfilePic],
	}, nil
}

func initLogger(logLevel string, logOutput string) {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "Jan _2 15:04:05.000000"
	customFormatter.FullTimestamp = true
	customFormatter.ForceColors = false
	customFormatter.DisableColors = true
	log.SetFormatter(customFormatter)
	err := os.Remove("http.txt")
	if err != nil {
		log.Error(err)
	}

	switch logLevel {
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		fallthrough
	default:
		log.SetLevel(log.InfoLevel)
	}

	switch logOutput {
	case "NONE":
		log.SetOutput(ioutil.Discard)
	case "FILE":
		file, err := os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Error(err)
		}
		log.SetOutput(file)
	case "ALL":
		file, err := os.OpenFile("http.txt", os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Error(err)
		}
		log.SetOutput(io.MultiWriter(file, os.Stdout))
	case "STDERR":
		fallthrough
	default:
		// Stderr is set by default
	}
}

func initPool() pool.Pool {
	myPool := new(pool.TcpPool).NewTcpPool(pool.TcpPoolConfig{
		InitialSize: 1000,
		MaxSize:     1200,
		Factory: func() (net.Conn, error) {
			return net.Dial("tcp", "127.0.0.1:9999")
		},
	})
	return myPool
}

func (srv *HTTPServer) withRequestId(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(api.RequestIdHeader)
		if rid == "" {
			rid = uuid.NewV4().String()
			r.Header.Set(api.RequestIdHeader, rid)
		}
		handler(w, r)
	}
}

func (srv *HTTPServer) withSessValidation(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(api.RequestIdHeader)
		logger := log.WithFields(log.Fields{api.RequestId: rid})
		logger.Info("Start session validation")
		c, err := r.Cookie(auth.SESS_COOKIE_NAME)
		if err != nil {
			if err == http.ErrNoCookie {
				w.WriteHeader(http.StatusUnauthorized)
				renderTemplate(w, "login", "Unauthorised, please login")
				return
			}
			log.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			renderTemplate(w, "login", nil)
			return
		}

		sid := c.Value
		logger.Debug("Getting user of session ", sid)
		user, err := srv.getSession(sid, rid)

		if err != nil {
			// no such session, serve as usual
			log.Error(err)
			logger.Info("Serving with no session")
			handler.ServeHTTP(w, r)
		} else {
			// server with user
			logger.Info("Serving with session", user)
			handler.ServeHTTP(w, r.WithContext(newContext(r.Context(), user)))
		}
	}
}

// returns a new context that carries value u
func newContext(ctx context.Context, u *api.User) context.Context {
	return context.WithValue(ctx, CONTEXT_KEY, u)
}

// returns the User value stored in ctx, if any
func fromContext(ctx context.Context) (*api.User, bool) {
	u, ok := ctx.Value(CONTEXT_KEY).(*api.User)
	return u, ok
}

func (srv *HTTPServer) Start() {
	templates = template.Must(template.ParseGlob("templates/*.html"))
	initLogger(*logLevel, *logOutput)

	log.Info("HTTP server listening on port ", srv.Port)

	// metrics
	http.Handle("/metrics", promhttp.Handler())

	// have the server listen on required routes
	http.HandleFunc("/", srv.withRequestId(srv.rootHandler))
	http.HandleFunc("/login", srv.withRequestId(srv.loginHandler))
	http.HandleFunc("/logout", srv.withRequestId(srv.logoutHandler))
	http.HandleFunc("/home", srv.withSessValidation(srv.withRequestId(srv.homeHandler)))
	http.HandleFunc("/edit", srv.withSessValidation(srv.withRequestId(srv.editHandler)))
	http.HandleFunc("/register", srv.withRequestId(srv.registerHandler))
	http.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir("./images"))))
	server := &http.Server{
		Addr:         ":" + srv.Port,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func (srv *HTTPServer) Stop() {
	srv.TcpPool.PrintStats()
	log.Info("HTTP server stopped.")
}

func main() {
	flag.Parse()
	log.Info("LOGLEVEL: " + *logLevel)
	log.Info("LOGOUTPUT: " + *logOutput)

	server := HTTPServer{
		Hostname: "127.0.0.1",
		Port:     "8080",
		TcpPool:  initPool(),
	}

	defer server.Stop()
	server.Start()
}
