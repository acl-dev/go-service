//go:build linux || darwin
// +build linux darwin

package master

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	stateFd       = 5
	listenFdStart = 6
)

var (
	listenFdCount = 1
	confPath      string
	sockType      string
	services      string
	privilege     = false
	verbose       = false
	chrootOn      = false
)

type PreJailFunc func()
type InitFunc func()
type ExitFunc func()

var (
	preJailHandler PreJailFunc = nil
	initHandler    InitFunc    = nil
	exitHandler    ExitFunc    = nil
	doneChan                   = make(chan bool)
	connCount                  = 0
	connMutex      sync.RWMutex
	stopping       = false
	prepareCalled  = false
)

// from command args
var (
	Configure    string
	ServiceName  string
	ServiceType  string
	Verbose      bool
	Unprivileged bool
	Chroot       = false
	SocketCount  = 1

	Alone bool
)

// setOpenMax set the max opened file handles for current process which let
// the process can handle more connections.
func setOpenMax() {
	var rlim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		fmt.Println("Get rlimit error: " + err.Error())
		return
	}
	if rlim.Max <= 0 {
		rlim.Max = 100000
	}
	rlim.Cur = rlim.Max

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		fmt.Println("Set rlimit error: " + err.Error())
	}
}

// initFlags init the command args come from acl_master; the application should call
// flag.Parse() in its main function!
func initFlags() {
	// Just walk through all the internal args to avoid fatal error from flag parser,
	// but these flags will be ignored, because we'll use the flags parsed in parseArgs().

	flag.StringVar(&Configure, "f", "", "app configure file (internal)")
	flag.StringVar(&ServiceName, "n", "", "app service name (internal)")
	flag.StringVar(&ServiceType, "t", "sock", "app service type (internal)")
	flag.BoolVar(&Alone, "alone", false, "stand alone running (internal)")
	flag.BoolVar(&Verbose, "v", false, "app verbose (internal)")
	flag.BoolVar(&Unprivileged, "u", false, "app unprivileged (internal)")
	flag.BoolVar(&Chroot, "c", false, "app chroot (internal)")
	flag.IntVar(&SocketCount, "s", 1, "listen fd count (internal)")
	if Verbose || verbose {
		log.Println("service:", ServiceName, "conf:", Configure)
	}
}

func init() {
	initFlags()
	setOpenMax()
}

func parseArgs() {
	var n = len(os.Args)
	for i := 0; i < n; i++ {
		switch os.Args[i] {
		case "-f":
			i++
			if i < n {
				confPath = os.Args[i]
			}
		case "-n":
			i++
			if i < n {
				services = os.Args[i]
			}
		case "-t":
			i++
			if i < n {
				sockType = os.Args[i]
			}
		case "-v":
			verbose = true
		case "-u":
			privilege = true
		case "-c":
			chrootOn = true
		case "-s":
			i++
			if i < n {
				listenFdCount, _ = strconv.Atoi(os.Args[i])
			}
		}
	}

	log.Printf("ListenFdCount=%d, sockType=%s, services=%s",
		listenFdCount, sockType, services)
}

// Prepare this function can be called automatically in net_service.go or
// web_service.go to load configure, and it can also be canned in application's
// main function
func Prepare() {
	if prepareCalled {
		return
	} else {
		prepareCalled = true
	}

	parseArgs()
	loadConf(confPath)
}

func chroot() {
	if len(AppArgs) == 0 || !privilege || len(AppOwner) == 0 {
		return
	}

	u, err := user.Lookup(AppOwner)
	if err != nil {
		log.Printf("Lookup %s error %s", AppOwner, err)
	} else {
		gid, err := strconv.Atoi(u.Gid)
		if err != nil {
			log.Printf("Invalid gid=%s, %s", u.Gid, err)
		} else if err := syscall.Setgid(gid); err != nil {
			log.Printf("Setgid error %s", err)
		} else {
			log.Printf("Setgid ok")
		}

		uid, err := strconv.Atoi(u.Uid)
		if err != nil {
			log.Printf("Invalid uid=%s, %s", u.Uid, err)
		} else if err := syscall.Setuid(uid); err != nil {
			log.Printf("Setuid error %s", err)
		} else {
			log.Printf("Setuid ok")
		}
	}

	if chrootOn && len(AppRootDir) > 0 {
		// The system call chroot can't work correctly on Linux.
		// In golang issue 1435 from the Go source comments.
		// On linux Setuid and Setgid only affects the current thread,
		// not the process. This does not match what most callers expect
		// so we must return an error here rather than letting the caller
		// think that the call succeeded.
		// But I wrote a sample that using setuid and setgid after
		// creating some threads, thease threads' uid and gid were
		// changed to the uid or gid by calling setuid and setgid, why?
		err := syscall.Chroot(AppRootDir)
		if err != nil {
			log.Printf("Chroot error %s, path %s", err, AppRootDir)
		} else {
			log.Printf("Chroot ok, path %s", AppRootDir)
			err := syscall.Chdir("/")
			if err != nil {
				log.Printf("Chdir error %s", err)
			} else {
				log.Printf("Chdir ok")
			}
		}
	}
}

// GetListenersByAddrs In run alone mode, the application should give the
// listening addrs and call this function to listen the given addrs
func GetListenersByAddrs(addrs string) ([]net.Listener, error) {
	if len(addrs) == 0 {
		log.Println("No valid addrs for listening")
		return nil, errors.New("no valid addrs for listening")
	}

	// split addrs like "xxx.xxx.xxx.xxx:port; xxx.xxx.xxx.xxx:port"

	addrs = strings.Replace(addrs, " ", "", -1)
	addrs = strings.Replace(addrs, ",", ";", -1)
	addrs = strings.Replace(addrs, "|", ":", -1)
	tokens := strings.Split(addrs, ";")

	cfg := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			})
		},
	}
	listeners := []net.Listener(nil)
	for _, addr := range tokens {
		ln, err := cfg.Listen(context.Background(), "tcp", addr)
		if err == nil {
			listeners = append(listeners, ln)
			log.Printf("Listen %s ok\r\n", addr)
			continue
		}

		log.Println("Listen", addr, "error:", err)
	}

	if len(listeners) == 0 {
		return nil, errors.New("no listeners were created")
	}
	return listeners, nil
}

// GetListeners In acl_master daemon running mode, this function will be called
// to init the listener handles.
func GetListeners() ([]net.Listener, error) {
	listeners := []net.Listener(nil)
	for fd := listenFdStart; fd < listenFdStart+listenFdCount; fd++ {
		file := os.NewFile(uintptr(fd), "open one listen fd")
		if file == nil {
			log.Printf("os.NewFile failed for fd=%d", fd)
			continue
		}

		ln, err := net.FileListener(file)

		// fd will be dupped in FileListener, so we should close it
		// after the listener is created
		_ = file.Close()

		if err == nil {
			listeners = append(listeners, ln)
			log.Printf("add fd %d ok", fd)
			continue
		}
		log.Println(fmt.Sprintf("Create FileListener error=\"%s\", fd=%d", err, fd))
	}

	if len(listeners) == 0 {
		log.Println("No listener created!")
		return nil, errors.New("no listener created")
	} else {
		log.Printf("Listeners's len=%d\r\n", len(listeners))
	}
	return listeners, nil
}

func ServiceInit(addrs string, stopHandler func(bool)) ([]net.Listener, error) {
	Prepare()

	if preJailHandler != nil {
		preJailHandler()
	}

	chroot()

	if initHandler != nil {
		initHandler()
	}

	// if addrs not empty, alone mode will be used, or daemon mode be used

	var listeners []net.Listener
	var daemonMode bool

	// If alone is false and sockType has been set, we'll start the service
	// in daemon mode, else we'll start the service in alone mode. The sockType
	// is coming from the the master service framework.

	if !Alone && len(sockType) > 0 {
		var err error
		if AppReusePort && len(AppService) > 0 {
			listeners, err = GetListenersByAddrs(AppService)
		} else {
			listeners, err = GetListeners()
		}
		if err != nil {
			log.Println("GetListeners failed", err)
			return nil, err
		}
		daemonMode = true
	} else if len(addrs) > 0 {
		var err error
		listeners, err = GetListenersByAddrs(addrs)
		if err != nil {
			return nil, err
		}
		daemonMode = false
	} else {
		log.Println("addrs empty in alone running mode")
		return nil, errors.New("no addresses given in alone running mode")
	}

	if len(listeners) == 0 {
		log.Println("No listener available!")
		return nil, errors.New("no listener available")
	}

	// In daemon mode, the backend monitor fiber will be created for
	// monitoring the status with the acl_master framework. If disconnected
	// from acl_master, the current child process will exit.
	if daemonMode {
		go monitorMaster(listeners, nil, stopHandler)
	}
	return listeners, nil
}

// monitorMaster monitor the PIPE IPC between the current process and acl_master,
// when acl_master close thePIPE, the current process should exit after
// which has handled all its tasks
func monitorMaster(listeners []net.Listener,
	onStopHandler func(), stopHandler func(bool)) {

	file := os.NewFile(uintptr(stateFd), "")
	conn, err := net.FileConn(file)
	if err != nil {
		panic(fmt.Sprintf("FileConn error=%s", err))
	}

	log.Println("Waiting for master exiting ...")

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		log.Println("Disconnected from master", err)
	}

	stopping = true

	if onStopHandler != nil {
		onStopHandler()
	} else {
		// XXX: force stopping listen again
		for _, ln := range listeners {
			log.Println("Closing one listener ", ln.Addr())
			_ = ln.Close()
		}
	}

	var n, i int
	n = 0
	i = 0

	if AppQuickAbort {
		log.Println("app_quick_abort been set")
	} else {
		for {
			connMutex.RLock()
			if connCount <= 0 {
				connMutex.RUnlock()
				break
			}

			n = connCount
			connMutex.RUnlock()
			time.Sleep(time.Second) // sleep 1 second
			i++
			log.Printf("Exiting, clients=%d, sleep=%d seconds\r\n", n, i)
			if AppWaitLimit > 0 && i >= AppWaitLimit {
				log.Printf("Waiting too long >= %d", AppWaitLimit)
				break
			}
		}
	}

	log.Println("master service disconnected, exiting now")

	if stopHandler != nil {
		stopHandler(true)
	}
}

func ConnCountInc() {
	connMutex.Lock()
	connCount++
	connMutex.Unlock()
}

func ConnCountDec() {
	connMutex.Lock()
	connCount--
	connMutex.Unlock()
}

func ConnCountCur() int {
	connMutex.RLock()
	n := connCount
	connMutex.RUnlock()
	return n
}

func OnPreJail(handler PreJailFunc) {
	preJailHandler = handler
}

func OnInit(handler InitFunc) {
	initHandler = handler
}

func OnExit(handler ExitFunc) {
	exitHandler = handler
}
