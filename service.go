// +build linux darwin

package master

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"runtime"
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
	listenFdCount int = 1
	confPath      string
	sockType      string
	services      string
	privilege     bool = false
	verbose       bool = false
	chrootOn      bool = false
)

type PreJailFunc func()
type InitFunc func()
type ExitFunc func()

var (
	preJailHandler PreJailFunc = nil
	initHandler    InitFunc    = nil
	exitHandler    ExitFunc    = nil
	doneChan       chan bool   = make(chan bool)
	connCount      int         = 0
	connMutex      sync.RWMutex
	stopping       bool = false
	prepareCalled  bool = false
)

// from configure file of the app
var (
	AppConf       *Config
	MasterService  string
	MasterLogPath  string
	MasterOwner    string
	MasterArgs     string
	AppRootDir     string
	AppUseLimit    int    = 0
	AppIdleLimit   int    = 0
	AppQuickAbort  bool   = false
	AppWaitLimit   int    = 10
	AppAccessAllow string = "all"
	Appthreads     int    = 0

	TlsCertFile    string
	TlsKeyFile     string
)

// from command args
var (
	MasterConfigure    string
	MasterServiceName  string
	MasterServiceType  string
	MasterVerbose      bool
	MasterUnprivileged bool
	//	MasterChroot       bool
	MasterSocketCount int = 1

	Alone             bool
)

const Version string = "1.0.3"

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
	flag.StringVar(&MasterConfigure, "f", "", "app configure file")
	flag.StringVar(&MasterServiceName, "n", "", "app service name")
	flag.StringVar(&MasterServiceType, "t", "sock", "app service type")
	flag.BoolVar(&Alone, "alone", false, "stand alone running")
	flag.BoolVar(&MasterVerbose, "v", false, "app verbose")
	flag.BoolVar(&MasterUnprivileged, "u", false, "app unprivileged")
	//	flag.BoolVar(&MasterChroot, "c", false, "app chroot")
	flag.IntVar(&MasterSocketCount, "s", 1, "listen fd count")
}

func init() {
	initFlags()
	setOpenMax()
}

func parseArgs() {
	var n = len(os.Args)
	for i := 0; i < n; i++ {
		switch os.Args[i] {
		case "-s":
			i++
			if i < n {
				listenFdCount, _ = strconv.Atoi(os.Args[i])
				if listenFdCount <= 0 {
					listenFdCount = 1
				}
			}
		case "-f":
			i++
			if i < n {
				confPath = os.Args[i]
			}
		case "-t":
			i++
			if i < n {
				sockType = os.Args[i]
			}
		case "-n":
			i++
			if i < n {
				services = os.Args[i]
			}
		case "-u":
			privilege = true
		case "-v":
			verbose = true
		case "-c":
			chrootOn = true
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

	AppConf = new(Config)
	AppConf.InitConfig(confPath)

	MasterLogPath = AppConf.GetString("master_log")
	if len(MasterLogPath) > 0 {
		f, err := os.OpenFile(MasterLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("OpenFile %s error %s", MasterLogPath, err)
		} else {
			log.SetOutput(f)
			//log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
	}

	MasterService = AppConf.GetString("master_service")
	MasterOwner = AppConf.GetString("master_owner")
	MasterArgs = AppConf.GetString("master_args")

	AppRootDir = AppConf.GetString("app_queue_dir")
	AppUseLimit = AppConf.GetInt("app_use_limit")
	AppIdleLimit = AppConf.GetInt("app_idle_limit")
	AppQuickAbort = AppConf.GetBool("app_quick_abort")
	AppWaitLimit = AppConf.GetInt("app_wait_limit")
	AppAccessAllow = AppConf.GetString("app_access_allow")
	Appthreads = AppConf.GetInt("app_threads")
	if Appthreads > 0 {
		runtime.GOMAXPROCS(Appthreads)
	}

	TlsCertFile = AppConf.GetString("tls_cert_file")
	TlsKeyFile = AppConf.GetString("tls_key_file")

	log.Printf("Args: %s, AppAccessAllow: %s\r\n", MasterArgs, AppAccessAllow)
}

func chroot() {
	if len(MasterArgs) == 0 || !privilege || len(MasterOwner) == 0 {
		return
	}

	u, err := user.Lookup(MasterOwner)
	if err != nil {
		log.Printf("Lookup %s error %s", MasterOwner, err)
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
		// changed to the // uid or gid by calling setuid and setgid, why?
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

// GetListenersByAddrs In run alone mode, the application should give the listening addrs
// and call this function to listen the given addrs
func GetListenersByAddrs(addrs string) ([]net.Listener, error) {
	if len(addrs) == 0 {
		log.Println("No valid addrs for listening")
		return nil, errors.New("no valid addrs for listening")
	}

	// split addrs like "xxx.xxx.xxx.xxx:port; xxx.xxx.xxx.xxx:port"

	addrs = strings.Replace(addrs, " ", "", -1)
	addrs = strings.Replace(addrs, ",", ";", -1)
	tokens := strings.Split(addrs, ";")

	listeners := []net.Listener(nil)
	for _, addr := range tokens {
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			listeners = append(listeners, ln)
			log.Printf("Listen %s ok\r\n", addr)
			continue
		}

		log.Println("Listen", addr, "error:", err)
	}
	if len(listeners) == 0 {
		return nil, errors.New("no listeners were created!")
	}
	return listeners, nil
}

// GetListeners In acl_master daemon running mode, this function will be called to
// init the listener handles.
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
	
	if len(addrs) > 0 {
		var err error
		listeners, err = GetListenersByAddrs(addrs)
		if err != nil {
			return nil, err
		}
		daemonMode = false
	} else {
		var err error
		listeners, err = GetListeners()
		if err != nil {
			log.Println("GetListeners failed", err)
			return nil, err
		}
		daemonMode = true
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

	log.Println("Master disconnected, exiting now")

	if stopHandler != nil {
		stopHandler(true)
	}
}

func connCountInc() {
	connMutex.Lock()
	connCount++
	connMutex.Unlock()
}

func connCountDec() {
	connMutex.Lock()
	connCount--
	connMutex.Unlock()
}

func connCountCur() int {
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
