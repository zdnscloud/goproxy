package goproxy

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	errFailedAuth       = errors.New("failed authentication")
	errWrongMessageType = errors.New("wrong websocket message type")
)

type Authorizer func(req *http.Request) (clientKey string, authed bool, err error)
type ErrorWriter func(rw http.ResponseWriter, req *http.Request, code int, err error)

func DefaultErrorWriter(rw http.ResponseWriter, req *http.Request, code int, err error) {
	rw.Write([]byte(err.Error()))
	rw.WriteHeader(code)
}

type Server struct {
	authorizer  Authorizer
	errorWriter ErrorWriter
	sessions    *sessionManager
	peerLock    sync.Mutex
}

func New(auth Authorizer, errorWriter ErrorWriter) *Server {
	return &Server{
		authorizer:  auth,
		errorWriter: errorWriter,
		sessions:    newSessionManager(),
	}
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	clientKey, authed, err := s.auth(req)
	if err != nil {
		s.errorWriter(rw, req, 400, err)
		return
	}
	if !authed {
		s.errorWriter(rw, req, 401, errFailedAuth)
		return
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
		Error:            s.errorWriter,
	}

	wsConn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		s.errorWriter(rw, req, 400, errors.New("Error during upgrade for host"))
		return
	}

	session := s.sessions.add(clientKey, wsConn)
	defer s.sessions.remove(session)

	fmt.Printf("add client with key %v", clientKey)
	// Don't need to associate req.Context() to the Session, it will cancel otherwise
	_, err = session.Serve()
	if err != nil {
		// Hijacked so we can't write to the client
	}
}

func (s *Server) auth(req *http.Request) (clientKey string, authed bool, err error) {
	return s.authorizer(req)
}
