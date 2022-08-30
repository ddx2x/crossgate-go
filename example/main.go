package main

import (
	"context"
	"fmt"
	"net"

	"github.com/ddx2x/crossgate-go/register"
	"github.com/ddx2x/crossgate-go/service"
)

var _ service.IService = &Service{}

type Service struct{}

func (Service) Name() string {
	return "test"
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
	for range ctx.Done() {
	}
	return nil
}

func main() {

	s := Service{}

	if err := service.MakeService(s); err != nil {
		panic(err)
	}

}
