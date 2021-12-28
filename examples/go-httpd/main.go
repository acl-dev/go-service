package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/acl-dev/master-go"
)

func handler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("served", r.URL)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Server", "go-httpd")
	fmt.Fprintf(w, "Hello World!\r\n")
}

var (
	filePath = flag.String(
		"c",
		"dummy.cf",
		"congiure filePath",
	)

	listenAddrs = flag.String(
		"listen",
		"127.0.0.1:8880",
		"Listen addrs in alone running",
	)
)

func main() {
	flag.Parse()

	master.Prepare()

	http.HandleFunc("/", handler)

	var err error
	if master.Alone {
		fmt.Println("listen:", listenAddrs)
		err = master.WebAloneStart(*listenAddrs, nil)
	} else {
		// daemon mode in master framework
		err = master.WebDaemonStart(nil)
	}
	if err != nil {
		log.Println("Start webserver failed:", err)
	}
}
