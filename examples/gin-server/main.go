package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/acl-dev/go-service"
	"github.com/gin-gonic/gin"
)

var (
	listenAddrs string // the server's listening addrs in alone mode.
)

var (
	g sync.WaitGroup // Used to wait for service to stop.
)

// Start one fiber server for one listener which is a webserver.
func startServer(listener net.Listener) {
	go func() {
		defer g.Done()

		e := gin.New()
		e.GET("/test", func(c *gin.Context) {
			c.String(200, "hello world!\r\n")
		})
		e.GET("/", func(c *gin.Context) {
			c.String(200, "hello world!\r\n")
		})

		log.Printf("Listen on %s", listener.Addr())
		//_ = e.RunListener(listener)
		server := &http.Server{
			Handler: e,
			ConnState: func(conn net.Conn, state http.ConnState) {
				switch state {
				case http.StateNew:
					master.ConnCountInc()
					fmt.Printf("New connectioin from %s\r\n", conn.RemoteAddr())
					break
				case http.StateClosed:
					master.ConnCountDec()
					fmt.Printf("Close connectioin from %s\r\n", conn.RemoteAddr())
					break
				case http.StateHijacked:
					master.ConnCountDec()
					fmt.Printf("Close connectioin from %s\r\n", conn.RemoteAddr())
					break
				default:
					fmt.Printf("Other status=%d\r\n", int(state))
					break
				}
			},
		}
		server.Serve(listener)
	}()
}

func parseArgs() {
	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:28880, 127.0.0.1:28881",
		"Listen addr in alone running")

	flag.Parse()
}

func main() {
	fmt.Println("Current master-go version:", master.Version)

	// parse args from commandline or acl_master's exec
	parseArgs()

	var listener []net.Listener
	var err error

	// Get all listeners which may be created by acl_master in daemon mode
	// or created by the current process in alone mode; In daemon mode,
	// the listeners' fds were created by acl_master and transfered to the
	// children processes after fork/exec.

	listener, err = master.ServiceInit(listenAddrs)
	if err != nil {
		log.Println("Listen error:", err)
		return
	}

	log.Printf("ServiceType=%s, test_src=%s, test_bool=%t\r\n",
		master.ServiceType, master.AppConf.GetString("test_src"),
		master.AppConf.GetBool("test_bool"))

	// Add the listener fibers' count in sync waiting group.
	g.Add(len(listener))

	// Start fibers for each listener which is listening on different addrs.
	for _, ln := range listener {
		startServer(ln)
	}

	log.Println("gin-server: Wait for services stop ...")

	// Wait for all listeners to stop.
	g.Wait()

	log.Println("gin-server: All services stopped!")
}
