package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/acl-dev/master-go"
)

func rootHandler(w http.ResponseWriter, _ *http.Request) {
	//fmt.Println("served", r.URL)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Server", "go-httpd")
	_, _ = fmt.Fprintf(w, "Hello World for access root path!\r\n")
}

func testHandler(w http.ResponseWriter, _ *http.Request) {
	//fmt.Println("served", r.URL)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Server", "go-httpd")
	_, _ = fmt.Fprintf(w, "Hello World for access test path!\r\n")
}

var (
	listenAddrs = flag.String(
		"listen",
		"127.0.0.1:8880",
		"Listen addrs in alone running",
	)
)

func main() {
	fmt.Println("Current master-go version:", master.Version)

	flag.Parse()

	master.Prepare()

	fmt.Printf("ServiceType=%s, test_src=%s, test_bool=%t\r\n",
		master.ServiceType, master.AppConf.GetString("test_src"),
		master.AppConf.GetBool("test_bool"))

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/test", testHandler)

	if master.Alone {
		fmt.Println("listen:", listenAddrs)
	}

	/*
	err := master.WebServiceStart(*listenAddrs, nil)
	if err != nil {
		log.Println("Start webserver failed:", err)
	}
	*/

	service, err := master.WebServiceInit(*listenAddrs, nil);
	if err != nil {
		log.Println("Init webservice failed:", err)
	} else {
		service.AcceptHandler = func(conn net.Conn) {
			fmt.Printf("Connection from %s\r\n", conn.RemoteAddr())
		}
		service.CloseHandler = func(conn net.Conn) {
			fmt.Printf("Disconnect from %s\r\n", conn.RemoteAddr())
		}
		service.Run()
	}
}
