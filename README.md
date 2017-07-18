[![Circle CI](https://circleci.com/gh/acl-dev/go-master.svg?style=svg)](https://circleci.com/gh/acl-dev/go-master)


# go-master

go 语言开发的服务器模板，可与 acl_master 服务器框架深度集成。


## 安装

	go get -u github.com/acl-dev/go-master

## 使用

    package main

    import (
        "flag"
        "fmt"
        "log"
        "net"
        "os"

        "github.com/acl-dev/go-master"
    )

    func onAccept(conn net.Conn) {
        buf := make([]byte, 8192)
        for {
            n, err := conn.Read(buf)
            if err != nil {
                fmt.Println("read over", err)
                break
            }

            conn.Write(buf[0:n])
        }
    }

    func onClose(conn net.Conn) {
        log.Println("---client onClose---")
    }

    var (
        filePath   string
        listenAddr string
    )

    func main() {
        flag.StringVar(&filePath, "c", "dummy.cf", "configure filePath")
        flag.StringVar(&listenAddr, "listen", "127.0.0.1:8880", "listen addr in alone running")

        flag.Parse()

        master.Prepare()
        master.OnClose(onClose)
        master.OnAccept(onAccept)

        if master.Alone {
            addrs := make([]string, 1)
            if len(listenAddr) == 0 {
                panic("listenAddr null")
            }

            addrs = append(addrs, listenAddr)

            fmt.Printf("listen:")
            for _, addr := range addrs {
                fmt.Printf(" %s", addr)
            }
            fmt.Println()

            master.NetStart(addrs)
        } else {
            // daemon mode in master framework
            master.NetStart(nil)
        }
    }


更多请参考[examples](https://github.com/acl-dev/go-master/tree/master/examples/)
