package goproxy

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	PingWaitDuration  = time.Duration(10 * time.Second)
	PingWriteInterval = time.Duration(5 * time.Second)
	MaxRead           = 8192
)

type wsConn struct {
	*websocket.Conn
	sync.Mutex
}

func newWSConn(conn *websocket.Conn) *wsConn {
	w := &wsConn{
		Conn: conn,
	}
	w.setupDeadline()
	return w
}

func (w *wsConn) WriteMessage(messageType int, data []byte) error {
	w.Lock()
	defer w.Unlock()
	w.Conn.SetWriteDeadline(time.Now().Add(PingWaitDuration))
	return w.Conn.WriteMessage(messageType, data)
}

func (w *wsConn) setupDeadline() {
	w.SetReadDeadline(time.Now().Add(PingWaitDuration))
	w.SetPingHandler(func(string) error {
		w.Lock()
		w.WriteControl(websocket.PongMessage, []byte(""), time.Now().Add(time.Second))
		w.Unlock()
		return w.SetReadDeadline(time.Now().Add(PingWaitDuration))
	})
	w.SetPongHandler(func(string) error {
		return w.SetReadDeadline(time.Now().Add(PingWaitDuration))
	})
}
