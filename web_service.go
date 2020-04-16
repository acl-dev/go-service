package master

import (
	"context"
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

// start WEB service with the specified listening addrs
func WebStart(addrs string) {
	var listeners []net.Listener
	listeners, err := ServiceInit(addrs)
	if err != nil {
		panic("ServiceInit failed")
	}
	
	if len(addrs) > 0 {
		daemonMode = false
	} else {
		daemonMode = true
	}
	
	for _, ln := range listeners {
		go webServ(ln, daemonMode)
	}

	if daemonMode {
		go monitorMaster(listeners, onWebStop, webStop)
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
