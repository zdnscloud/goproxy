package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	counter int64
	listen  string
)

func main() {
	flag.StringVar(&listen, "listen", ":8125", "Listen address")
	flag.Parse()

	handler := func(rw http.ResponseWriter, req *http.Request) {
		next := atomic.AddInt64(&counter, 1)
		fmt.Println("request", next)

		time.Sleep(time.Duration(rand.Intn(300)) * time.Millisecond)
		rw.Write([]byte("HI"))
	}
	http.HandleFunc("/counter", handler)
	fmt.Println("listening on:", listen)
	http.ListenAndServe(listen, nil)
}
