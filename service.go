package master

import (
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

// set the max opened file handles for current process which let
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

// init the command args come from acl_master; the application should call
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

// this function can be called automatically in net_service.go or
// web_service.go to load configure, and it can also be canned in appliction's
// main function
func Prepare() {
	if prepareCalled {
		return
	} else {
		prepareCalled = true
	}

	parseArgs()

	conf := new(Config)
	conf.InitConfig(confPath)

	MasterLogPath = conf.GetString("master_log")
	if len(MasterLogPath) > 0 {
		f, err := os.OpenFile(MasterLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("OpenFile %s error %s", MasterLogPath, err)
		} else {
			log.SetOutput(f)
			//log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
	}

	MasterService = conf.GetString("master_service")
	MasterOwner = conf.GetString("master_owner")
	MasterArgs = conf.GetString("master_args")

	AppRootDir = conf.GetString("app_queue_dir")
	AppUseLimit = conf.GetInt("app_use_limit")
	AppIdleLimit = conf.GetInt("app_idle_limit")
	AppQuickAbort = conf.GetBool("app_quick_abort")
	AppWaitLimit = conf.GetInt("app_wait_limit")
	AppAccessAllow = conf.GetString("app_access_allow")
	Appthreads = conf.GetInt("app_threads")
	if Appthreads > 0 {
		runtime.GOMAXPROCS(Appthreads)
	}

	log.Printf("Args: %s, AppAccessAllow: %s\r\n", MasterArgs, AppAccessAllow)
}

func chroot() {
	if len(MasterArgs) == 0 || !privilege || len(MasterOwner) == 0 {
		return
	}

	user, err := user.Lookup(MasterOwner)
	if err != nil {
		log.Printf("Lookup %s error %s", MasterOwner, err)
	} else {
		gid, err := strconv.Atoi(user.Gid)
		if err != nil {
			log.Printf("Invalid gid=%s, %s", user.Gid, err)
		} else if err := syscall.Setgid(gid); err != nil {
			log.Printf("Setgid error %s", err)
		} else {
			log.Printf("Setgid ok")
		}

		uid, err := strconv.Atoi(user.Uid)
		if err != nil {
			log.Printf("Invalid uid=%s, %s", user.Uid, err)
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

// In run alone mode, the application should give the listening addrs and call
// this function to listen the given addrs
func GetListenersByAddrs(addrs string) []net.Listener {
	if len(addrs) == 0 {
		panic("No valid addrs for listening")
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
			log.Printf("Listen %s ok", addr)
			continue
		}

		panic(fmt.Sprintf("Listen error=\"%s\", addr=%s", err, addr))
	}
	return listeners
}

// In acl_master daemon running mode, this function will be called for init
// the listener handles.
func GetListeners() []net.Listener {
	listeners := []net.Listener(nil)
	for fd := listenFdStart; fd < listenFdStart+listenFdCount; fd++ {
		file := os.NewFile(uintptr(fd), "open one listenfd")
		if file == nil {
			log.Printf("os.NewFile failed for fd=%d", fd)
			continue
		}

		// fd will be dupped in FileListener, so we should close it
		// after the listener is created
		defer file.Close()

		ln, err := net.FileListener(file)
		if err == nil {
			listeners = append(listeners, ln)
			log.Printf("add fd %d ok", fd)
			continue
		}
		log.Println(fmt.Sprintf("Create FileListener error=\"%s\", fd=%d", err, fd))
	}

	if len(listeners) == 0 {
		panic("No listener created!")
	} else {
		log.Printf("Listeners's len=%d", len(listeners))
	}
	return listeners
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
		listeners = GetListenersByAddrs(addrs)
		daemonMode = false
	} else {
		listeners = GetListeners()
		daemonMode = true
	}

	if len(listeners) == 0 {
		panic("No available listener!")
	}

	// In daemon mode, the backend monitor fiber will be created for
	// monitoring the status with the acl_master framework. If disconnected
	// from acl_master, the current child process will exit.
	if daemonMode {
		go monitorMaster(listeners, nil, stopHandler)
	}
	return listeners, nil
}

// monitor the PIPE IPC between the current process and acl_master,
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
			ln.Close()
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
