package register

import (
	"context"

	"github.com/ddx2x/crossgate-go/plugin"
)

const Default_LoadBalancer_Algorithm = RoundRobin

type LoadBalancerAlgorithm = string

const (
	RoundRobin = "round_robin"
	Random     = "random"
	Strict     = "strict"
)

func Register(ctx context.Context, p plugin.Plugin, name string, lba LoadBalancerAlgorithm, addr string, typ uint8) error {
	content := plugin.Content{
		Service: name,
		Lba:     lba,
		Addr:    addr,
		Type:    typ,
	}
	if err := p.Set(ctx, name, content); err != nil {
		return err
	}
	return nil
}
