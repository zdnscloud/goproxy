package goproxy

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type sessionListener interface {
	sessionAdded(clientKey string, sessionKey int64)
	sessionRemoved(clientKey string, sessionKey int64)
}

type sessionManager struct {
	sync.Mutex
	clients   map[string][]*Session
	listeners map[sessionListener]bool
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		clients:   map[string][]*Session{},
		listeners: map[sessionListener]bool{},
	}
}

func toDialer(s *Session, prefix string, deadline time.Duration) Dialer {
	return func(proto, address string) (net.Conn, error) {
		if prefix == "" {
			return s.serverConnect(deadline, proto, address)
		}
		return s.serverConnect(deadline, prefix+"::"+proto, address)
	}
}

func (sm *sessionManager) removeListener(listener sessionListener) {
	sm.Lock()
	defer sm.Unlock()

	delete(sm.listeners, listener)
}

func (sm *sessionManager) addListener(listener sessionListener) {
	sm.Lock()
	defer sm.Unlock()

	sm.listeners[listener] = true

	for k, sessions := range sm.clients {
		for _, session := range sessions {
			listener.sessionAdded(k, session.sessionKey)
		}
	}
}

func (sm *sessionManager) getDialer(clientKey string, deadline time.Duration) (Dialer, error) {
	sm.Lock()
	defer sm.Unlock()

	sessions := sm.clients[clientKey]
	if len(sessions) > 0 {
		return toDialer(sessions[0], "", deadline), nil
	}
	return nil, fmt.Errorf("failed to find Session for client %s", clientKey)
}

func (sm *sessionManager) add(clientKey string, conn *websocket.Conn) *Session {
	sessionKey := rand.Int63()
	session := newSession(sessionKey, clientKey, conn)

	sm.Lock()
	defer sm.Unlock()

	sm.clients[clientKey] = append(sm.clients[clientKey], session)

	for l := range sm.listeners {
		l.sessionAdded(clientKey, session.sessionKey)
	}

	return session
}

func (sm *sessionManager) remove(s *Session) {
	sm.Lock()
	defer sm.Unlock()

	store := sm.clients
	var newSessions []*Session
	for _, v := range store[s.clientKey] {
		if v.sessionKey == s.sessionKey {
			continue
		}
		newSessions = append(newSessions, v)
	}

	if len(newSessions) == 0 {
		delete(store, s.clientKey)
	} else {
		store[s.clientKey] = newSessions
	}

	for l := range sm.listeners {
		l.sessionRemoved(s.clientKey, s.sessionKey)
	}

	s.Close()
}
