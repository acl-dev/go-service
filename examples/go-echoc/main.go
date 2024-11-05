package main

import (
	"flag"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var (
	serverAddr = "127.0.0.1:28881"
	connsCount = 100
	ioCount    = 10000
	cpusNum    = 1
)

func main() {
	flag.StringVar(&serverAddr, "server", serverAddr, "server address")
	flag.IntVar(&connsCount, "conns", connsCount, "number of connections")
	flag.IntVar(&ioCount, "io", ioCount, "number of io count")
	flag.IntVar(&cpusNum, "cpus", cpusNum, "number of cpus")
	flag.Parse()

	if cpusNum > 0 {
		runtime.GOMAXPROCS(cpusNum)
	}

	var wg sync.WaitGroup
	wg.Add(connsCount)

	var count int64 = 0
	begin := time.Now()

	for i := 0; i < connsCount; i++ {
		go func() {
			defer wg.Done()

			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				fmt.Println("Connect " + serverAddr + " error:" + err.Error())
				return
			}
			defer conn.Close()

			fmt.Println("Connect " + serverAddr + " success")
			buf := make([]byte, 1024)

			for j := 0; j < ioCount; j++ {
				data := "hello world!"
				if _, err := conn.Write([]byte(data)); err != nil {
					fmt.Println("Write " + serverAddr + " error:" + err.Error())
					return
				}
				if _, err := conn.Read(buf); err != nil {
					fmt.Println("Read " + serverAddr + " error:" + err.Error())
					return
				}
				atomic.AddInt64(&count, 1)
			}
		}()
	}

	wg.Wait()
	end := time.Now()
	tc := end.Sub(begin)
	var speed float64
	if tc > 0 {
		speed = (float64(count) * 1000) / float64(tc.Milliseconds())
	} else {
		speed = 10000000000000
	}
	fmt.Printf("time cost: %d ms, count: %d, speed: %.2f\r\n",
		tc.Milliseconds(), count, speed)
}
