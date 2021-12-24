package main

import (
	"flag"
	"log"
	"net"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/acl-dev/master-go"
)

var (
	filePath    string	// the server's configure file.
	listenAddrs string	// the server's listening addrs in alone mode.
)

var (
	g sync.WaitGroup	// Used to wait for service to stop.
)

// Start one fiber server for one listener which is a webserver.
func startServer(listener net.Listener)  {
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
		e.RunListener(listener)
	}()
}

func parseArgs()  {
	flag.StringVar(&filePath, "c", "gin-server.cf", "Configure filePath")
	flag.StringVar(&listenAddrs, "listen", "127.0.0.1:28880, 127.0.0.1:28881",
		"Listen addr in alone running")

	flag.Parse()
}

func main()  {
	// parse args from commandline or acl_master's exec
	parseArgs()

	var listener []net.Listener
	var err error

	// Get all listeners which may be created by acl_master in daemon mode
	// or created by the current process in alone mode; In daemon mode,
	// the listeners' fds were created by acl_master and transfered to the
	// children processes after fork/exec.

	if master.Alone {
		listener, err = master.ServiceInit(listenAddrs, nil)
	} else {
		listener, err = master.ServiceInit("", onStop)
	}
	if err != nil {
		log.Println("Listen error:", err)
		return
	}

	// Add the listener fibers' count in sync waiting group.
	g.Add(len(listener))

	// Start fibers for each listener which is listening on different addrs.
	for _, ln := range listener {
		startServer(ln)
	}

	log.Println("gin-server: Wait for services stop ...")

	// Wait for all listeners to stop.
	g.Wait()

	log.Println("gin-server: All services stopped!\r\n")
}

// The callback used in master.ServiceInit() for the second arg, when the server
// is to being stopped, the callback will be called.
func onStop(ok bool) {
	log.Println("gin-server: Disconnect from acl_master!")
}
