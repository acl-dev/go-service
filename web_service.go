package master

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
)

var (
	daemonMode bool = false
	webServers []http.Server
)

func webServ(ln net.Listener, daemon bool) {
	serv := http.Server{}
	if daemon {
		webServers = append(webServers, serv)
	}

	serv.Serve(ln)
}

func WebAloneStart(addrs string) error {
	if len(addrs) == 0 {
		log.Println("addrs empty")
		return errors.New("addrs empty")
	}
	return webStart(addrs)
}

func WebDaemonStart() error{
	return webStart("")
}

// start WEB service with the specified listening addrs
func webStart(addrs string) error {
	var listeners []net.Listener
	listeners, err := ServiceInit(addrs, webStop)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return err
	}

	var daemonMode bool
	if len(addrs) > 0 {
		daemonMode = false
	} else {
		daemonMode = true
	}

	for _, ln := range listeners {
		// create fiber for each listener to accept connections
		go webServ(ln, daemonMode)
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

func onWebStop() {
	var wg sync.WaitGroup
	wg.Add(len(webServers))
	for _, ln := range webServers {
		go func() {
			defer wg.Done()
			ln.Shutdown(context.Background())
		}()
	}
	wg.Wait()
}

func webStop(n bool) {
	if doneChan != nil {
		doneChan <- n
	}
}
