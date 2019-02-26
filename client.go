package goproxy

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
)

type ConnectAuthorizer func(proto, address string) bool

func ClientConnect(wsURL string, headers http.Header, dialer *websocket.Dialer, auth ConnectAuthorizer, onConnect func(context.Context) error) {
	connectToProxy(wsURL, headers, auth, dialer, onConnect)
}

func connectToProxy(proxyURL string, headers http.Header, auth ConnectAuthorizer, dialer *websocket.Dialer, onConnect func(context.Context) error) error {
	if dialer == nil {
		dialer = &websocket.Dialer{}
	}
	ws, _, err := dialer.Dial(proxyURL, headers)
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
