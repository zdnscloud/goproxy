// +build !windows

package main

import (
	"flag"
	"fmt"

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

	goproxy.RegisterAgent(fmt.Sprintf("%s/%s", addr, id), func(string, string) bool { return true }, nil)
}
