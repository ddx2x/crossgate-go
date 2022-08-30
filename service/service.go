package service

import (
	"context"
	"sync"

	"github.com/ddx2x/crossgate-go/plugin"
	"github.com/ddx2x/crossgate-go/register"
)

type IService interface {
	Name() string
	Addr() string
	Lba() string

	Start(ctx context.Context) error
}

func MakeService(ctx context.Context, ss ...IService) error {
	init_env()
	
	wg := &sync.WaitGroup{}
	ec := make(chan error)
	for _, s := range ss {
		p, err := plugin.Get(ctx, wg, plugin.MongoDBPlugin, get_register_addr())
		if err != nil {
			return err
		}
		if err := register.Register(ctx, p, s.Name(), s.Lba(), s.Addr()); err != nil {
			return err
		}
		go func(s IService) {
			ec <- s.Start(ctx)
		}(s)
	}
	wg.Wait()

	return <-ec
}
