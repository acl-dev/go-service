package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/acl-dev/master-go"
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
	filePath    string
	listenAddrs string
)

func main() {
	flag.StringVar(&filePath, "c", "dummy.cf", "configure filePath")
	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:28880, 127.0.0.1:28881",
		"listen addr in alone running")

	flag.Parse()

	master.Prepare()

	fmt.Printf("filePath=%s, MasterServiceType=%s\r\n",
		filePath, master.MasterServiceType)

	master.OnClose(onClose)
	master.OnAccept(onAccept)

	if master.Alone {
		fmt.Printf("listen: %s\r\n", listenAddrs)
		master.TcpStart(listenAddrs)
	} else {
		// daemon mode in master framework
		master.TcpStart("")
	}
}
