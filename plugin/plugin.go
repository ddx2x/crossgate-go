package plugin

import (
	"context"
	"sync"
)

type PluginType = uint8

const (
	MongoDBPlugin PluginType = iota
)

type Content struct {
	Service string `json:"service" bson:"service"`
	Lba     string `json:"lba" bson:"lba"`
	Addr    string `json:"addr" bson:"addr"`
	Type    uint8  `json:"type" bson:"type"`
}

type Plugin interface {
	Set(ctx context.Context, name string, value Content) error
}

func Get(ctx context.Context, wg *sync.WaitGroup, tpy PluginType, uri string) (Plugin, error) {
	switch tpy {
	case MongoDBPlugin:
		return initMongoPlugin(ctx, wg, uri)
	}
	return initMongoPlugin(ctx, wg, uri)
}
