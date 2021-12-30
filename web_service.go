package master

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	daemonMode = false
	webServers []*http.Server
)

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return true
}

func webServ(ln net.Listener, daemon bool, handler http.Handler) {
	serv := &http.Server{ Handler: handler }
	if daemon {
		webServers = append(webServers, serv)
	}

	if len(TlsCertFile) > 0 && len(TlsKeyFile) > 0 &&
		pathExist(TlsCertFile) && pathExist(TlsKeyFile) {

		_ = serv.ServeTLS(ln, TlsCertFile, TlsKeyFile)
	} else {
		_ = serv.Serve(ln)
	}
}

func WebAloneStart(addrs string, handler http.Handler) error {
	if len(addrs) == 0 {
		log.Println("addrs empty")
		return errors.New("addrs empty")
	}
	return WebStart(addrs, handler)
}

func WebDaemonStart(handler http.Handler) error{
	return WebStart("", handler)
}

// WebStart start WEB service with the specified listening addrs
func WebStart(addrs string, handler http.Handler) error {
	var listeners []net.Listener
	listeners, err := ServiceInit(addrs, webStop)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return err
	}

	if len(addrs) > 0 {
		daemonMode = false
	} else {
		daemonMode = true
	}

	for _, ln := range listeners {
		// create fiber for each listener to accept connections
		go webServ(ln, daemonMode, handler)
	}

	log.Println("service started!")
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

/*
func onWebStop() {
	var wg sync.WaitGroup

	wg.Add(len(webServers))

	for _, ln := range webServers {
		ln := ln
		go func() {
			defer wg.Done()
			_ = ln.Shutdown(context.Background())
		}()
	}

	wg.Wait()
}
*/

func webStop(n bool) {
	if doneChan != nil {
		doneChan <- n
	}
}
