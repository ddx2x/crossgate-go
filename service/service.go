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

	Start(ctx context.Context) error
}

func MakeService(ctx context.Context, ss ...IService) error {
	init_env()

	wg := &sync.WaitGroup{}
	ec := make(chan error)
	service_names := make([]string, 0)
	for _, s := range ss {
		service_names = append(service_names, strings.Split(s.Name(), ",")...)

		for _, n := range service_names {
			p, err := plugin.Get(ctx, wg, plugin.MongoDBPlugin, get_register_addr())
			if err != nil {
				return err
			}
			if err := register.Register(ctx, p, n, s.Lba(), s.Addr()); err != nil {
				return err
			}
		}
		go func(s IService) {
			ec <- s.Start(ctx)
		}(s)
	}
	wg.Wait()

	return <-ec
}
