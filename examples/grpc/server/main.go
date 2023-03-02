package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/acl-dev/go-service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"strconv"
	"sync"

	"grpc/pb"
)

var (
	listenAddrs = ":8885" // the server's listening addrs in alone mode.
)

var (
	g sync.WaitGroup // Used to wait for service to stop.
)

type server struct{}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	name := "Hello "
	name += in.Name

	message := "your message: "
	if in.Message != nil {
		message += *in.Message
	}

	return &pb.HelloReply{Name: &name, Message: &message}, nil
}

func (s *server) GetMessage(ctx context.Context, in *pb.MessageRequest) (*pb.MessageReply, error) {
	message := "your message: "
	message += in.Message

	message += ", your age: "
	message += strconv.Itoa(int(in.Age))

	if in.Min != nil {
		message += ", your min: "
		message += strconv.Itoa(int(*in.Min))
	}

	return &pb.MessageReply{Message: message}, nil
}

func startServer(ln net.Listener) {
	fmt.Println("Listen on:", ln.Addr())

	go func() {
		defer g.Done()

		s := grpc.NewServer()
		pb.RegisterGreetsServer(s, &server{})
		reflection.Register(s)
		if err := s.Serve(ln); err != nil {
			fmt.Println("start server error", err)
		}
	}()
}

func main() {
	flag.Parse()
	master.Prepare()

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

	// Add the listener fibers' count in sync waiting group.
	g.Add(len(listener))

	// Start fibers for each listener which is listening on different addrs.
	for _, ln := range listener {
		startServer(ln)
	}

	log.Println("grpc server: Wait for services stop ...")

	// Wait for all listeners to stop.
	g.Wait()

	log.Println("grpc server: All services stopped!")
}
