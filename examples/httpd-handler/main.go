package main

import (
	"flag"
	"fmt"
	"github.com/acl-dev/master-go"
	"log"
	"net/http"
)

var (
	filePath    string
	listenAddrs string
)

type MyHandler map[string]float32

func (handler MyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/list":
		for item, price := range handler {
			_, _ = fmt.Fprintf(w, "%s: %.2f\r\n", item, price)
		}
	case "/price":
		item := req.URL.Query().Get("item")
		if len(item) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, "no item in request url: %s", req.URL)
			return
		}

		price, ok := handler[item]
		if ok {
			_, _ = fmt.Fprintf(w, "%s: %.2f dollars\r\n", item, price)
		} else {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, "no such item: %s\r\n", item)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, "no such page: %s\r\n", req.URL)
	}
}

func main() {
	fmt.Println("Current master-go version:", master.Version)

	flag.StringVar(&filePath, "c", "dummy.cf", "configure filePath")
	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:8088", "listen addr in alone running")
	flag.Parse()

	master.Prepare()

	handler := MyHandler{ "shoes": 50, "socks": 5 }
	var err error

	if master.Alone {
		fmt.Println("listen:", listenAddrs)
		err = master.WebAloneStart(listenAddrs, handler)
	} else {
		// daemon mode in master framework
		err = master.WebDaemonStart(handler)
	}
	if err != nil {
		log.Println("Start webserver failed:", err)
	}
}
