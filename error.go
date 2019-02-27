package goproxy

import (
	"errors"
)

var (
	errFailedAuth           = errors.New("failed authentication")
	errWrongMessageType     = errors.New("wrong websocket message type")
	errConnectionBufferFull = errors.New("connection buffer is full")
)
