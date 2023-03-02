package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"github.com/acl-dev/go-service"
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

func onClose(conn net.Conn) {
	log.Println("---client onClose---", conn.RemoteAddr())
}

var (
	listenAddrs string
)

func main() {
	fmt.Println("Current master-go version:", master.Version)

	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:28880, 127.0.0.1:28881",
		"listen addr in alone running")

	// Parse the commandline args.
	flag.Parse()

	// Parse commandline args needed internal, load configure info.
	master.Prepare()

	fmt.Printf("ServiceType=%s, test_src=%s, test_bool=%t\r\n",
		master.ServiceType, master.AppConf.GetString("test_src"),
		master.AppConf.GetBool("test_bool"))

	/*
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
    */

	// Bind the given addresses from commandline or from master framework.
	service, err := master.TcpServiceInit(listenAddrs)
	if err != nil {
		log.Println("Init tcp service error:", err)
		return
	}

	// Set callback when accepting one connection.
	service.AcceptHandler = onAccept

	// Set callback when closing one connection.
	service.CloseHandler = onClose

	fmt.Printf("listen: %s\r\n", listenAddrs)

	// Start the service in alone or daemon mode.
	service.Run()
}
