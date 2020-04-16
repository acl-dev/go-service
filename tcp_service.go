package master

import (
	//"flag"
	"log"
	"net"
	"os"
	"time"
)

type AcceptFunc func(net.Conn)
type CloseFunc func(net.Conn)

var (
	acceptHandler AcceptFunc = nil
	closeHandler  CloseFunc  = nil
)

func handleConn(conn net.Conn) {
	if acceptHandler == nil {
		panic("acceptHandler nil")
	}

	connCountInc()

	acceptHandler(conn)

	if closeHandler != nil {
		closeHandler(conn)
	} else {
		log.Println("closeHandler nil")
	}

	conn.Close()

	connCountDec()
}

func loopAccept(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error", err)
			time.Sleep(1000 * time.Millisecond)
			continue
		}

		go handleConn(conn)
	}

	if stopping {
		log.Println("server stopping")
	} else {
		log.Println("server listen error")
		tcpStop(false)
	}
}

func OnAccept(handler AcceptFunc) {
	for _, arg := range os.Args {
		log.Println("arg=", arg)
	}

	acceptHandler = handler
}

func OnClose(handler CloseFunc) {
	closeHandler = handler
}

// start TCP service with the specified listening addrs
func TcpStart(addrs string) {
	var daemon bool
	var listeners []net.Listener
	listeners, err := ServiceInit(addrs)
	if err != nil {
		panic("ServiceInit failed")
	}

	if len(addrs) > 0 {
		daemon = false
	} else {
		daemon = true
	}

	for _, ln := range listeners {
		// create fiber for each listener to accept connections
		go loopAccept(ln)
	}

	// if in daemon mode, the backend monitor fiber will be created for
	// monitoring the status with the acl_master framework

	if daemon {
		go monitorMaster(listeners, nil, tcpStop)
	}

	log.Println("service started!")

	// waiting for service stopped
	res := <-doneChan

	close(doneChan)
	doneChan = nil

	if exitHandler != nil {
		exitHandler()
	}

	if res {
		log.Println("service stopped normal!")
	} else {
		log.Println("service stopped abnormal!")
	}
}

// callback when service stopped, be called in service.go
func tcpStop(ok bool) {
	if doneChan != nil {
		// notify the main fiber to exit now
		doneChan <- ok
	}
}
