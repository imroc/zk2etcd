package main

import (
	flag "github.com/spf13/pflag"
	"strings"
)

type Common struct {
	zookeeperPrefix        string
	zookeeperExcludePrefix string // TODO: 先简单实现 exclude，后续优化 exlude 判断的性能
}

func (c *Common) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.zookeeperPrefix, "zookeeper-prefix", "/dubbo", "comma-separated list of zookeeper path prefix to be synced")
	fs.StringVar(&c.zookeeperExcludePrefix, "zookeeper-exclude-prefix", "/dubbo/config", "comma-separated list of zookeeper path prefix to be excluded")
}

func (c *Common) GetAll() (zkPrefixes, zkExcludePrefixes []string) {
	zkPrefixes = strings.Split(c.zookeeperPrefix, ",")
	if len(zkPrefixes) == 0 {
		zkPrefixes = append(zkPrefixes, "/")
	}
	zkExcludePrefixes = strings.Split(c.zookeeperExcludePrefix, ",")

	return
}
