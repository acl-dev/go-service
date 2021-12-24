package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"github.com/acl-dev/master-go"
)

var (
	filePath    string
	listenAddrs string
)

type MyHandler map[string]float32

func (self MyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/list":
		for item, price := range self {
			fmt.Fprintf(w, "%s: %.2f\r\n", item, price)
		}
	case "/price":
		item := req.URL.Query().Get("item")
		if len(item) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "no item in request url: %s", req.URL)
			return
		}

		price, ok := self[item]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "no such item: %s\r\n", item)
		} else {
			fmt.Fprintf(w, "%s: %.2f dollars\r\n", item, price)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "no such page: %s\r\n", req.URL)
	}
}

func main() {
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
