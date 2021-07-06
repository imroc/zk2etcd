package record

import (
	flag "github.com/spf13/pflag"
)

var Enable bool

type Options struct {
	RedisServer string
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.RedisServer, "redis-server", "", "redis server address")
}

func (o *Options) build() *Record {
	return New(o.RedisServer)
}

func Init(opt *Options) {
	if opt.RedisServer == "" {
		return
	}
	Enable = true
	defaultRecord = opt.build()
}
