package service

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ddx2x/crossgate-go/plugin"
	"github.com/ddx2x/crossgate-go/register"
)

type IService interface {
	Name() string
	Addr() string
	Lba() string

	Start(ctx context.Context) error
}

func MakeService(s IService) error {
	init_env()
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		<-c
		cancel()
	}()

	wg := &sync.WaitGroup{}
	p, err := plugin.Get(ctx, wg, plugin.MongoDBPlugin, get_register_addr())
	if err != nil {
		return err
	}

	if err := register.Register(ctx, p, s.Name(), s.Lba(), s.Addr()); err != nil {
		return err
	}

	if err := s.Start(ctx); err != nil {
		return err
	}

	wg.Wait()

	return nil
}
