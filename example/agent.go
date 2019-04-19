// +build !windows

package main

import (
	"flag"
	"fmt"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/goproxy"
)

var (
	addr string
	id   string
)

func main() {
	flag.StringVar(&addr, "url", "ws://localhost:8123/register", "Address to connect to")
	flag.StringVar(&id, "id", "foo", "Client ID")
	flag.Parse()

	log.InitLogger(log.Debug)

	err := goproxy.RegisterAgent(fmt.Sprintf("%s/%s", addr, id), func(string, string) bool { return true }, nil)
	if err != nil {
		log.Errorf("agent get err:%s", err.Error())
	}
}
