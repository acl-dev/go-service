package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/acl-dev/go-master"
)

func onAccept(conn net.Conn) {
	buf := make([]byte, 8192)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Println("read over", err)
			break
		}

		conn.Write(buf[0:n])
	}
}

func onClose(conn net.Conn) {
	log.Println("---client onClose---")
}

var (
	filePath   string
	runAlone   bool
	listenAddr string
)

func main() {
	flag.StringVar(&filePath, "c", "dummy.cf", "configure filePath")
	flag.BoolVar(&runAlone, "alone", false, "stand alone running")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:8880", "listen addr in alone running")
	flag.Parse()

	fmt.Printf("filePath=%s, MasterServiceType=%s\r\n", filePath, master.MasterServiceType)

	master.OnClose(onClose)
	master.OnAccept(onAccept)

	if runAlone {
		addrs := make([]string, 1)
		if len(listenAddr) == 0 {
			panic("listenAddr null")
		}

		addrs = append(addrs, listenAddr)

		fmt.Printf("listen:")
		for _, addr := range addrs {
			fmt.Printf(" %s", addr)
		}
		fmt.Println()

		master.NetStart(addrs)
	} else {
		// daemon mode in master framework
		master.NetStart(nil)
	}
}
