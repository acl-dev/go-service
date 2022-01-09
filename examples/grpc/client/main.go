package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
	"time"

	"grpc/pb"
)

const (
	address = "127.0.0.1:8885"
)

var (
	name    = flag.String("N", "zsxxsz", "user name")
	nclient = flag.Int("c", 1, "nclient count")
	nloop   = flag.Int("n", 100, "nloop count")
)

func main()  {
	flag.Parse()

	var okCount int64

	var wg sync.WaitGroup
	wg.Add(*nclient)

	begin := time.Now()

	for i := 0; i < *nclient; i++ {
		go func() {
			defer wg.Done()
			doClient(*nloop, &okCount)
		}()
	}

	wg.Wait()

	cost := time.Now().Sub(begin)
	speed := (okCount * 1000) / cost.Milliseconds()
	fmt.Println("ok count:", okCount, ", cost:", cost, ", speed:", speed)
}

func doClient(nloop int, okcount *int64)  {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		fmt.Println("connect", address, "error", err)
		return
	}

	defer conn.Close()

	message := "hello"
	c := pb.NewGreetsClient(conn)

	for i := 0; i < nloop; i++ {
		m := &pb.HelloRequest{ Name: *name, Message: &message }
		//ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
		//defer cancel()

		r, err := c.SayHello(context.Background(), m)
		if err != nil {
			fmt.Println("rpc SayHello error:", err)
			return
		}

		atomic.AddInt64(okcount, 1)
		if i < 1 {
			fmt.Println("Greet reply:", *r.Name, ",", *r.Message)
		}
	}

	for i := 0; i < nloop; i++ {
		m := &pb.MessageRequest{ Message: message, Age: int32(i + 1) }
		r, err := c.GetMessage(context.Background(), m)
		if err != nil {
			fmt.Println("rpc GetMessage error:", err)
			return
		}

		atomic.AddInt64(okcount, 1)
		if i < 1 {
			fmt.Println("GetMessage reply:", r.Message)
		}
	}
}
