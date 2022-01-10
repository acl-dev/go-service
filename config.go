package master

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const Version string = "1.0.8"

type Config struct {
	Entries map[string]string
}

// from configure file of the app
var (
	AppConf         *Config
	AppService       string
	AppLogPath       string
	AppOwner         string
	AppArgs          string
	AppRootDir       string
	AppUseLimit    = 0
	AppIdleLimit   = 0
	AppQuickAbort  = false
	AppWaitLimit   = 10
	AppAccessAllow = "all"
	Appthreads     = 0

	TlsCertFile      string
	TlsKeyFile       string
)

func loadConf(confPath string) {
	AppConf = new(Config)
	AppConf.InitConfig(confPath)

	AppLogPath = AppConf.GetString("master_log")
	if len(AppLogPath) > 0 {
		f, err := os.OpenFile(AppLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0643)
		if err != nil {
			fmt.Printf("open %s error %s\r\n", AppLogPath, err.Error())
		} else {
			log.SetOutput(f)
			//log.SetOutput(io.MultiWriter(os.Stderr, f))
		}
	}

	AppService = AppConf.GetString("master_service")
	AppOwner = AppConf.GetString("master_owner")
	AppArgs = AppConf.GetString("master_args")

	AppRootDir = AppConf.GetString("app_queue_dir")
	AppUseLimit = AppConf.GetInt("app_use_limit")
	AppIdleLimit = AppConf.GetInt("app_idle_limit")
	AppQuickAbort = AppConf.GetBool("app_quick_abort")
	AppWaitLimit = AppConf.GetInt("app_wait_limit")
	AppAccessAllow = AppConf.GetString("app_access_allow")
	Appthreads = AppConf.GetInt("app_threads")
	if Appthreads > -1 {
		runtime.GOMAXPROCS(Appthreads)
	}

	TlsCertFile = AppConf.GetString("tls_cert_file")
	TlsKeyFile = AppConf.GetString("tls_key_file")

	log.Printf("AppArgs: %s, AppAccessAllow: %s\r\n", AppArgs, AppAccessAllow)
}

func (c *Config) InitConfig(path string) {
	c.Entries = make(map[string]string)

	if len(path) == 0 {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	r := bufio.NewReader(f)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		s := strings.TrimSpace(string(line))
		eq := strings.Index(s, "=")
		if eq < 0 {
			continue
		}

		name := strings.TrimSpace(s[:eq])
		if len(name) == 0 {
			continue
		}

		value := strings.TrimSpace(s[eq+1:])

		pos := strings.Index(value, "\t#")
		if pos > -1 {
			value = value[0:pos]
		}

		pos = strings.Index(value, " #")
		if pos > -1 {
			value = value[0:pos]
		}

		if len(value) == 0 {
			continue
		}

		c.Entries[name] = strings.TrimSpace(value)
	}
}

func (c Config) GetString(name string) string {
	val, found := c.Entries[name]
	if !found {
		return ""
	}
	return val
}

func (c Config) GetInt(name string) int {
	val, found := c.Entries[name]
	if !found {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	} else {
		return n
	}
}

func (c Config) GetBool(name string) bool {
	val, found := c.Entries[name]
	if !found {
		return false
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return false
	} else {
		return n != 0
	}
}
