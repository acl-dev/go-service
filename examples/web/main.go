package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/acl-dev/master-go"
)

func handler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("served", r.URL)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Server", "web")
	fmt.Fprintf(w, "Hello World!\r\n")
}

var (
	filePath    string
	listenAddrs string
)

func main() {
	flag.StringVar(&filePath, "c", "dummy.cf", "configure filePath")
	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:8880", "listen addr in alone running")
	flag.Parse()

	master.Prepare()

	http.HandleFunc("/", handler)

	if master.Alone {
		fmt.Println("listen: ", listenAddrs)
		master.WebStart(listenAddrs)
	} else {
		// daemon mode in master framework
		master.WebStart("")
	}
}
