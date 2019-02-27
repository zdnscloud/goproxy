package goproxy

import (
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Session struct {
	sync.Mutex

	nextConnID int64
	conn       *wsConn
	conns      map[int64]*connection
	auth       ConnectAuthorizer
	client     bool
}

func NewClientSession(auth ConnectAuthorizer, conn *websocket.Conn) *Session {
	return &Session{
		nextConnID: 1,
		conn:       newWSConn(conn),
		conns:      map[int64]*connection{},
		auth:       auth,
		client:     true,
	}
}

func newSession(agentKey string, conn *websocket.Conn) *Session {
	return &Session{
		nextConnID: 1,
		conn:       newWSConn(conn),
		conns:      map[int64]*connection{},
	}
}

func (s *Session) Serve() (int, error) {
	if s.client {
		s.conn.startPing()
	}

	for {
		typ, reader, err := s.conn.NextReader()
		if err != nil {
			return 400, err
		}

		if typ != websocket.BinaryMessage {
			return 400, errWrongMessageType
		}

		if err := s.serveMessage(reader); err != nil {
			return 500, err
		}
	}
}

func (s *Session) serveMessage(reader io.Reader) error {
	message, err := newServerMessage(reader)
	if err != nil {
		return err
	}

	if message.messageType == Connect {
		if s.auth == nil || !s.auth(message.proto, message.address) {
			return errors.New("connect not allowed")
		}
		s.clientConnect(message)
		return nil
	}

	s.Lock()
	conn := s.conns[message.connID]
	s.Unlock()

	if conn == nil {
		if message.messageType == Data {
			newErrorMessage(message.connID, err).WriteTo(s.conn)
		}
		return nil
	}

	switch message.messageType {
	case Data:
		if _, err := conn.WriteMessage(message); err != nil {
			conn.reportErr(err)
			s.closeConnection(message.connID)
		}
	case Error:
		s.closeConnection(message.connID)
	}

	return nil
}

func (s *Session) closeConnection(connID int64) {
	s.Lock()
	conn := s.conns[connID]
	delete(s.conns, connID)
	s.Unlock()

	conn.Close()
}

func (s *Session) clientConnect(message *message) {
	conn := newConnection(message.connID, s, message.proto, message.address)
	s.Lock()
	s.conns[message.connID] = conn
	s.Unlock()
	go proxyRealService(conn, message)
}

func (s *Session) getDialer(deadline time.Duration) Dialer {
	return func(proto, address string) (net.Conn, error) {
		return s.createConnectionForClient(deadline, proto, address)
	}
}

func (s *Session) createConnectionForClient(deadline time.Duration, proto, address string) (net.Conn, error) {
	connID := atomic.AddInt64(&s.nextConnID, 1)
	conn := newConnection(connID, s, proto, address)

	s.Lock()
	s.conns[connID] = conn
	s.Unlock()

	_, err := s.writeMessage(newConnect(connID, deadline, proto, address))
	if err != nil {
		conn.reportErr(err)
		s.closeConnection(connID)
		return nil, err
	}

	return conn, err
}

func (s *Session) writeMessage(message *message) (int, error) {
	return message.WriteTo(s.conn)
}

func (s *Session) Close() {
	s.Lock()
	defer s.Unlock()

	if s.client {
		s.conn.stopPing()
	}

	for _, conn := range s.conns {
		conn.Close()
	}
	s.conns = map[int64]*connection{}
}
