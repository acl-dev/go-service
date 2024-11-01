package main

import (
	"flag"
	"fmt"
	"github.com/acl-dev/go-service"
	"log"
	"net"
	"runtime"
	"time"
)

var (
	listenAddrs  string
	readTimeout  = -1
	writeTimeout = -1
	numCPUs      = -1
)

func onAccept(conn net.Conn) {
	buf := make([]byte, 8192)
	for {
		if readTimeout > 0 {
			err := conn.SetReadDeadline(time.Now().Add(time.Duration(readTimeout) * time.Second))
			if err != nil {
				return
			}
		}
		if writeTimeout > 0 {
			err := conn.SetWriteDeadline(time.Now().Add(time.Duration(writeTimeout) * time.Second))
			if err != nil {
				return
			}
		}
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

func main() {
	fmt.Println("Current go-service's version:", master.Version)

	flag.IntVar(&numCPUs, "cpus", runtime.NumCPU(), "Number of CPUs to use")
	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:28880, 127.0.0.1:28881",
		"listen addr in alone running")
	flag.IntVar(&readTimeout, "r", -1, "read timeout")
	flag.IntVar(&writeTimeout, "w", -1, "write timeout")

	// Parse the commandline args.
	flag.Parse()

	if numCPUs > 0 {
		runtime.GOMAXPROCS(numCPUs)
	}

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
