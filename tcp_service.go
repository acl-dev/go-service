package master

import (
	"errors"
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

func TcpAloneStart(addrs string) error {
	if len(addrs) == 0 {
		log.Println("Addrs empty")
		return errors.New("Addrs empty")
	}
	return TcpStart(addrs)
}

func TcpDaemonStart() error {
	return TcpStart("")
}

// start TCP service with the specified listening addrs
func TcpStart(addrs string) error {
	var listeners []net.Listener
	listeners, err := ServiceInit(addrs, tcpStop)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return err
	}

	for _, ln := range listeners {
		// create fiber for each listener to accept connections
		go loopAccept(ln)
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
	return nil
}

// callback when service stopped, be called in service.go
func tcpStop(ok bool) {
	if doneChan != nil {
		// notify the main fiber to exit now
		doneChan <- ok
	}
}
