package master

import (
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

type WebService struct {
	listeners     []net.Listener
	webServs      []*http.Server
	handler       http.Handler
	AcceptHandler AcceptFunc
	CloseHandler  CloseFunc
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return true
}

func (service *WebService) webServ(ln net.Listener) {
	serv := &http.Server{
		Handler: service.handler,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				ConnCountInc()
				if service.AcceptHandler != nil {
					service.AcceptHandler(conn)
				}
			case http.StateActive:
			case http.StateIdle:
			case http.StateClosed, http.StateHijacked:
				ConnCountDec()
				if service.CloseHandler != nil {
					service.CloseHandler(conn)
				}
			default:
			}
		},
	}

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
	var g sync.WaitGroup // Used to wait for service to stop.

	g.Add(len(service.listeners))

	for _, ln := range service.listeners {
		// Create fiber for each listener to accept client connection.
		go func(l net.Listener) {
			defer g.Done()

			service.webServ(l)
		}(ln)
	}

	log.Println("webservice started!")

	// Call Wait() in service.go to wait the end of the service.
	res := Wait()

	// Waiting all the web listening services done.
	g.Wait()

	if exitHandler != nil {
		exitHandler()
	}

	if res {
		log.Printf("pid=%d: webservice stopped normal!\r\n", os.Getpid())
	} else {
		log.Printf("pid=%d, webservice stopped abnormal!\r\n", os.Getpid())
	}
}

func WebServiceInit(addrs string, handler http.Handler) (*WebService, error) {
	listeners, err := ServiceInit(addrs)
	if err != nil {
		log.Println("ServiceInit failed:", err)
		return nil, err
	}

	return &WebService{listeners: listeners, handler: handler}, nil
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
