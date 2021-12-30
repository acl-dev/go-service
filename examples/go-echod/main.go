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

		_, err = conn.Write(buf[0:n])
		if err != nil {
			fmt.Println("write error", err)
			break
		}
	}
}

func onClose(net.Conn) {
	log.Println("---client onClose---")
}

var (
	listenAddrs string
)

func main() {
	fmt.Println("Current master-go version:", master.Version)

	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:28880, 127.0.0.1:28881",
		"listen addr in alone running")

	flag.Parse()

	master.Prepare()

	fmt.Printf("MasterServiceType=%s\r\n", master.MasterServiceType)

	master.OnClose(onClose)
	master.OnAccept(onAccept)

	var err error
	if master.Alone {
		fmt.Printf("listen: %s\r\n", listenAddrs)
		err = master.TcpAloneStart(listenAddrs)
	} else {
		// daemon mode in master framework
		err = master.TcpDaemonStart()
	}
	if err != nil {
		log.Println("start tcp server error:", err)
	}
}
