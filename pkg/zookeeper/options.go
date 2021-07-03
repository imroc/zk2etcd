package zookeeper

import (
	flag "github.com/spf13/pflag"
	"strings"
)

var client *Client

func Init(o *Options) {
	client = o.buildClient()
}

type Options struct {
	ZookeeperServers string
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.ZookeeperServers, "zookeeper-servers", "", "comma-separated list of zookeeper servers address")
}

func (o *Options) buildClient() *Client {
	servers := strings.Split(o.ZookeeperServers, ",")
	return NewClient(servers)
}
