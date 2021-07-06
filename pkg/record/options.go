package record

import (
	flag "github.com/spf13/pflag"
)

type Options struct {
	RedisServer string
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.RedisServer, "redis-server", "127.0.0.1:6379", "redis server address")
}

func (o *Options) build() *Record {
	return New(o.RedisServer)
}

func Init(opt *Options) {
	defaultRecord = opt.build()
}
