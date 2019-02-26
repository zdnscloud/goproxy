package goproxy

import (
	"context"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"net/http"
	"time"
)

type ConnectAuthorizer func(proto, address string) bool

func ClientConnect(wsURL string, headers http.Header, dialer *websocket.Dialer, auth ConnectAuthorizer, onConnect func(context.Context) error) error {
	if dialer == nil {
		dialer = &websocket.Dialer{}
	}
	ws, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return err
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if onConnect != nil {
		if err := onConnect(ctx); err != nil {
			return err
		}
	}

	session := NewClientSession(auth, ws)
	_, err = session.Serve()
	session.Close()
	return err
}

func proxyRealService(dialer Dialer, conn *connection, message *message) {
	defer conn.Close()
	netConn, err := net.DialTimeout(message.proto, message.address, time.Duration(message.deadline)*time.Millisecond)
	if err != nil {
		conn.writeErr(err)
		return
	}
	pipe(conn, netConn)
}

func pipe(client *connection, server net.Conn) {
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(server, client)
		errCh <- err
	}()

	go func() {
		_, err := io.Copy(client, server)
		errCh <- err
	}()
	err := <-errCh
	server.Close()
	client.writeErr(err)
}
