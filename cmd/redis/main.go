package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/record"
)

func main() {
	opt := &record.Options{
		RedisServer: "127.0.0.1:6379",
	}
	record.Init(opt)
	//fmt.Println(record.Get("/dubbo/douyu/test3"))
	fmt.Println(record.Put("/dubbo/douyu/test3", "douyudouyu3"))
}
