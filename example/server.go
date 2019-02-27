package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/zdnscloud/goproxy"
)

var (
	clients = map[string]*http.Client{}
	l       sync.Mutex
)

func authorizer(req *http.Request) (string, bool, error) {
	vars := mux.Vars(req)
	agentKey := vars["id"]
	return agentKey, agentKey != "", nil
}

func Client(server *goproxy.Server, rw http.ResponseWriter, req *http.Request) {
	timeout := req.URL.Query().Get("timeout")
	if timeout == "" {
		timeout = "15"
	}

	vars := mux.Vars(req)
	agentKey := vars["id"]
	url := fmt.Sprintf("%s://%s%s", vars["scheme"], vars["host"], vars["path"])
	client := getClient(server, agentKey, timeout)

	resp, err := client.Get(url)
	if err != nil {
		rw.Write([]byte(err.Error()))
		rw.WriteHeader(500)
		return
	}
	defer resp.Body.Close()

	rw.WriteHeader(resp.StatusCode)
	io.Copy(rw, resp.Body)
}

func getClient(server *goproxy.Server, agentKey, timeout string) *http.Client {
	l.Lock()
	defer l.Unlock()

	key := fmt.Sprintf("%s/%s", agentKey, timeout)
	client := clients[key]
	if client != nil {
		return client
	}

	dialer := server.GetAgentDialer(agentKey, 15*time.Second)
	client = &http.Client{
		Transport: &http.Transport{
			Dial: dialer,
		},
	}
	if timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err == nil {
			client.Timeout = time.Duration(t) * time.Second
		}
	}

	clients[key] = client
	return client
}

func main() {
	var (
		addr  string
		debug bool
	)
	flag.StringVar(&addr, "listen", ":8123", "Listen address")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	handler := goproxy.New(authorizer)
	router := mux.NewRouter()
	router.Handle("/register/{id}", handler)
	router.HandleFunc("/agent/{id}/{scheme}/{host}{path:.*}", func(rw http.ResponseWriter, req *http.Request) {
		Client(handler, rw, req)
	})

	fmt.Println("Listening on ", addr)
	http.ListenAndServe(addr, router)
}
