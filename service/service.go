package service

import (
	"context"
	"strings"
	"sync"

	"github.com/ddx2x/crossgate-go/plugin"
	"github.com/ddx2x/crossgate-go/register"
)

type IService interface {
	Name() string
	Addr() string
	Lba() string
	Type() uint8

	Start(ctx context.Context) error
}

func MakeService(ctx context.Context, ss ...IService) error {
	init_env()

	wg := &sync.WaitGroup{}
	ec := make(chan error)
	for _, s := range ss {
		for _, n := range strings.Split(s.Name(), ",") {
			p, err := plugin.Get(ctx, wg, plugin.MongoDBPlugin, get_register_addr())
			if err != nil {
				return err
			}
			if err := register.Register(ctx, p, n, s.Lba(), s.Addr(), 1); err != nil {
				return err
			}
		}
		go func(s IService, ec chan error) {
			ec <- s.Start(ctx)
		}(s, ec)
	}

	wg.Wait()

	return <-ec
}
