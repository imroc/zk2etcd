package record

import (
	flag "github.com/spf13/pflag"
)

var Enable bool

type Options struct {
	RedisServer   string
	RedisPassword string
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.RedisServer, "redis-server", "", "redis server address")
	fs.StringVar(&o.RedisPassword, "redis-password", "", "redis password")
}

func (o *Options) build() *Record {
	return New(o.RedisServer, o.RedisPassword)
}

func Init(opt *Options) {
	if opt.RedisServer == "" {
		return
	}
	Enable = true
	defaultRecord = opt.build()
}
