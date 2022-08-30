package service

import (
	"os"

	"github.com/ddx2x/crossgate-go/plugin"
	"github.com/spf13/pflag"
)

var register_type *string

func get_register_type() plugin.PluginType {
	rt := os.Getenv("REGISTER_TYPE")
	if rt == "" {
		rt = *register_type
	}

	switch rt {
	case "mongodb":
		return plugin.MongoDBPlugin
	}

	return plugin.MongoDBPlugin
}

var register_addr *string

func get_register_addr() string {
	if os.Getenv("REGISTER_ADDR") != "" {
		return os.Getenv("REGISTER_ADDR")
	}
	return *register_addr
}

func init_env() {
	register_addr = pflag.String("REGISTER_ADDR", "", "--REGISTER_ADDR=mongodb://127.0.0.1:27017")
	register_type = pflag.String("REGISTER_TYPE", "mongodb", "--REGISTER_TYPE=mongodb")
	pflag.Parse()
}
