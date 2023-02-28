package master

import (
	"errors"
	"log"
	"net"
	"os"
	"time"
)

type AcceptFunc func(net.Conn)
type CloseFunc func(net.Conn)

type TcpService struct {
	AcceptHandler AcceptFunc
	CloseHandler  CloseFunc

	listeners []net.Listener
}

func (service *TcpService) handleConn(conn net.Conn) {
	if service.AcceptHandler == nil {
		panic("acceptHandler nil")
	}

	ConnCountInc()

	service.AcceptHandler(conn)

	if service.CloseHandler != nil {
		service.CloseHandler(conn)
	}

	_ = conn.Close()

	ConnCountDec()
}

func (service *TcpService) loopAccept(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error", err)
			time.Sleep(1000 * time.Millisecond)
			break
		}

		go service.handleConn(conn)
	}

	if stopping {
		log.Println("server stopping")
	} else {
		log.Println("server listen error")
		tcpStop(false)
	}
}

func (service *TcpService) Run()  {
	for _, ln := range service.listeners {
		// create fiber for each listener to accept connections
		go service.loopAccept(ln)
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

func TcpServiceInit(addrs string) (*TcpService, error) {
	listeners, err := ServiceInit(addrs, tcpStop)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return nil, err
	}
	return &TcpService{ listeners: listeners }, nil
}

var (
	acceptHandler AcceptFunc = nil
	closeHandler  CloseFunc  = nil
)

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
	return TcpServiceStart(addrs)
}

func TcpDaemonStart() error {
	return TcpServiceStart("")
}

// TcpStart start TCP service with the specified listening addrs
func TcpServiceStart(addrs string) error {
	service, err := TcpServiceInit(addrs)
	if err != nil {
		return err
	}

	service.CloseHandler = closeHandler
	service.AcceptHandler = acceptHandler
	service.Run()
	return nil
}

// callback when service stopped, be called in service.go
func tcpStop(ok bool) {
	if doneChan != nil {
		// notify the main fiber to exit now
		doneChan <- ok
	}
}
