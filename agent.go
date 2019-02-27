package goproxy

import (
	"context"
	//"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
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

func proxyRealService(conn *connection, message *message) {
	netConn, err := net.DialTimeout(message.proto, message.address, time.Duration(message.deadline)*time.Millisecond)
	if err != nil {
		conn.reportErr(err)
		return
	}
	pipe(conn, netConn)
}

func pipe(client *connection, server net.Conn) {
	client.wg.Add(2)
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(server, client)
		errCh <- err
		client.wg.Done()
	}()

	go func() {
		_, err := io.Copy(client, server)
		errCh <- err
		client.wg.Done()
	}()

	err := <-errCh
	server.Close()
	client.reportErr(err)
}
