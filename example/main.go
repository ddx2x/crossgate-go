package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ddx2x/crossgate-go/register"
	"github.com/ddx2x/crossgate-go/service"
)

var _ service.IService = &Service{}

type Service struct {
	Flag string
}

func (s Service) Name() string {
	return s.Flag
}

func (Service) Addr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return fmt.Sprintf("%s:%s", ipnet.IP.String(), "3000")
			}
		}
	}
	return ""
}

func (Service) Lba() string {
	return register.Default_LoadBalancer_Algorithm
}

func (Service) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		<-c
		cancel()
	}()

	if err := service.MakeService(ctx, []service.IService{Service{Flag: "test1"}, Service{"test2"}}); err != nil {
		panic(err)
	}

}
