[![Circle CI](https://circleci.com/gh/acl-dev/master-go.svg?style=svg)](https://circleci.com/gh/acl-dev/master-go)

# master-go

go 语言开发的服务器模板，可与 acl_master 服务器框架深度集成。

* [一、安装](#一安装)
* [二、使用](#二使用)
    * [2.1、简单示例](#21简单示例)
    * [2.2、将 Go 服务程序部署在 acl_master 框架下](#22将-go-服务程序部署在-acl_master-框架下)

        * [2.2.1 部署 acl_master 服务管理框架](#221-部署-acl_master-服务管理框架)
        * [2.2.2 部署 Go 服务程序至 acl_master 框架下](#222-部署-go-服务程序至-acl_master-框架下)
* [三、参考](#三参考)
## 一、安装
```
	go get -u github.com/acl-dev/master-go
```

## 二、使用

### 2.1、简单示例
编写源码 `main.go` 如下：
```go
package main

import (
    "flag"
    "fmt"
    "log"
    "net"

    "github.com/acl-dev/master-go"
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
    filePath    string
    listenAddrs string
)

func main() {
    flag.StringVar(&filePath, "c", "dummy.cf", "configure filePath")
    flag.StringVar(&listenAddrs, "listen", "127.0.0.1:8080; 127.0.0.1:8081", "listen addr in alone running")

    flag.Parse()

    master.Prepare()
    master.OnClose(onClose)
    master.OnAccept(onAccept)

    if master.Alone {
        // run alone
        fmt.Printf("listen: %s\r\n", listenAddrs)
        master.TcpStart(listenAddrs)
    } else {
        // daemon mode in acl_master framework
        master.TcpStart("")
    }
}
```
编译：
```
$ go build -o echod
```
手工运行：

```
$ ./echod -alone
```
该程序为一个简单的回显服务，运行后可以手工 `telneet 127.0.0.1 8080` 进行测试。 

### 2.2、将 Go 服务程序部署在 acl_master 框架下
#### 2.2.1 部署 acl_master 服务管理框架
首先需要从 `https://github.com/acl-dev/acl` 下载 acl 工程，然后编译安装，过程如下：
```
#cd acl; make
#cd disk/master; ./setup.sh /opt/soft/acl-master
#cd /opt/soft/acl-master/sh; ./start.sh
```
上面过程便完成了编译、安装及启动 acl_master 服务管理框架的过程。  
如果您使用 CentOS 操作系统，还可以通过下面过程来完成（即：生成 acl_master RPM 包，然后安装该 RPM 包即可）：
```
#cd packaging; make
#cd x86_64; rpm -ivh acl-master*.rpm
```
当 RPM 安装后 acl_master 服务管理程序会自动启动。

#### 2.2.2 部署 Go 服务程序至 acl_master 框架下
首先下载 master-go 软件包并编译其中的服务示例，然后安装这些服务程序：

```
#go get -u github.com/acl-dev/master-go
#cd master-go/examples/
#(cd go-echod; go build; ./setup.sh /opt/soft/go-echod)
#(cd go-httpd; go build; ./setup.sh /opt/soft/go-httpd)
#(cd gin-server; go get; go build; ./setup.sh /opt/soft/gin-server)
#/opt/soft/go-echod/bin/start.sh
#/opt/soft/go-httpd/bin/start.sh
#/opt/soft/gin-server/bin/start.sh
```
通过启动脚本分别启动这几个服务例子，启动脚本实际上是通知 **`acl_master`** 服务程序来启动这几个服务程序。

最后运行 **`acl_master`** 服务框架中的管理工具来查看由 **`acl_master`** 管理的服务：
```
#/opt/soft/acl-master/bin/master-ctl -a list
```
结果显示如下：

```
status  service                                         type    proc    owner   conf    
200     127.0.0.1|5001, 127.0.0.1|5002, echod.sock      4       1       root    /opt/soft/go-echod/conf/go-echod.cf
200     |8881, 127.0.0.1|8882, go-httpd.sock            4       2       nobody  /opt/soft/go-httpd/conf/go-httpd.cf
200     |87, |88, |89, gin-server.sock                  4       2       nobody  /opt/soft/gin-server/conf/gin-server.cf
```
说明 **`acl_master`** 服务管理程序已经管理了这几个 Go 写的服务进程。同时可以看到：
- `gin-server` 服务由 `acl_master` 启动了两个进程；
- `gin-server` 可以同时多个 TCP 端口地址（其实监听行为是由 `acl_master` 执行的，`gin-server` 只是继成了这些监听行为）；
- `gin-server` 不仅可以监听 TCP 端口，还可同时监听 UNIX 域地址（这是由 acl_master 监听后传递给 gin-server 的）；
- `gin-server` 虽然监听的三个 TCP 端口号 < 1024，但 `gin-server` 的运行身份已经被 `acl_master` 切换为普通身份（nobody）。

此外，还需要两点需要注明：
- 在 UNIX 系统平台上，服务程序监听的端口如果小于 1024，则操作系统则要求此时的运行身份需为 root；
- 在 Linux 平台下 Go 语言编写的服务程序是无法正常切换运行身份的（具体可参考 Go 源码）。  

在 **`acl_master`** 的运行机制下，可以完美解决以上二者的矛盾。

## 三、参考
更多请参考 [examples](https://github.com/acl-dev/master-go/tree/master/examples/)
