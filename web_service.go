package master

import (
	"log"
	"net"
	"net/http"
	"os"
)

type WebService struct {
	listeners []net.Listener
	webServs []*http.Server
	handler http.Handler
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return true
}

func (service *WebService) webServ(ln net.Listener) {
	serv := &http.Server{ Handler: service.handler }
	service.webServs = append(service.webServs, serv)

	if len(TlsCertFile) > 0 && len(TlsKeyFile) > 0 &&
		pathExist(TlsCertFile) && pathExist(TlsKeyFile) {

		_ = serv.ServeTLS(ln, TlsCertFile, TlsKeyFile)
	} else {
		_ = serv.Serve(ln)
	}
}

// WebServiceStart start WEB service with the specified listening addrs
func (service *WebService) Run() {
	for _, ln := range service.listeners {
		// create fiber for each listener to accept connections
		go service.webServ(ln)
	}

	log.Println("webservice started!")
	res := <-doneChan

	close(doneChan)
	doneChan = nil

	if exitHandler != nil {
		exitHandler()
	}

	if res {
		log.Println("webservice stopped normal!")
	} else {
		log.Println("webservice stopped abnormal!")
	}
}

func WebServiceInit(addrs string, handler http.Handler) (*WebService, error) {
	listeners, err := ServiceInit(addrs, webStop)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return nil, err
	}

	return &WebService { listeners: listeners, handler: handler }, nil
}

// WebServiceStart start WEB service with the specified listening addrs
func WebServiceStart(addrs string, handler http.Handler) error {
	service, err := WebServiceInit(addrs, handler)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return err
	}

	service.Run()
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
