package goproxy

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type sessionManager struct {
	sync.Mutex
	clients map[string]*Session
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		clients: make(map[string]*Session),
	}
}

func toDialer(s *Session, deadline time.Duration) Dialer {
	return func(proto, address string) (net.Conn, error) {
		return s.serverConnect(deadline, proto, address)
	}
}

func (sm *sessionManager) getDialer(clientKey string, deadline time.Duration) (Dialer, error) {
	sm.Lock()
	defer sm.Unlock()

	if session, ok := sm.clients[clientKey]; ok {
		return toDialer(session, deadline), nil
	} else {
		return nil, fmt.Errorf("failed to find Session for client %s", clientKey)
	}
}

func (sm *sessionManager) add(clientKey string, conn *websocket.Conn) (*Session, error) {
	sm.Lock()
	defer sm.Unlock()
	if _, ok := sm.clients[clientKey]; ok {
		return nil, fmt.Errorf("duplicate agent key %s", clientKey)
	}

	session := newSession(clientKey, conn)
	sm.clients[clientKey] = session
	return session, nil
}

func (sm *sessionManager) remove(s *Session) {
	sm.Lock()
	defer sm.Unlock()

	delete(sm.clients, s.clientKey)
	s.Close()
}
