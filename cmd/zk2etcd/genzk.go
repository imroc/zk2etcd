package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"github.com/spf13/cobra"
	"go.uber.org/atomic"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func newGenzkCmd(args []string) *cobra.Command {
	var concurrent, parentCount, childrenCount uint
	var zookeeperServers string
	var path string
	var keyPrefix string
	var keyCount atomic.Int32
	var wg sync.WaitGroup

	cmd := &cobra.Command{
		Use:   "genzk",
		Short: "generate zookeeper keys for test",

		DisableAutoGenTag: true,

		PreRun: func(cmd *cobra.Command, args []string) {
			opt := &log.Option{LogLevel: "debug"}
			log.Init(opt)
		},

		Run: func(cmd *cobra.Command, args []string) {
			zkClient := zookeeper.NewClient(strings.Split(zookeeperServers, ","))
			ch := make(chan string, concurrent)
			go func() {
				for i := uint(0); i < parentCount; i++ {
					key := filepath.Join(path, keyPrefix+strconv.Itoa(int(i)))
					wg.Add(1)
					ch <- key
				}
			}()
			go func() {
				for range time.Tick(1 * time.Second) {
					fmt.Println("written key count:", keyCount.String())
				}
			}()
			for i := uint(0); i < concurrent; i++ {
				go func() {
					for k := range ch {
						zkClient.Create(k)
						for i := uint(0); i < childrenCount; i++ {
							key := keyPrefix + strconv.Itoa(int(i))
							zkClient.Create(filepath.Join(k, key))
							keyCount.Inc()
						}
						wg.Done()
					}
				}()
			}
			time.Sleep(1 * time.Second)
			wg.Wait()
		},
	}

	cmd.SetArgs(args)
	cmd.Flags().StringVar(&keyPrefix, "keyPrefix", "test", "key prefix to be generated")
	cmd.Flags().UintVar(&childrenCount, "children-count", 200, "how many child of each parent")
	cmd.Flags().UintVar(&parentCount, "parent-count", 200, "how many parent")
	cmd.Flags().StringVar(&path, "path", "/dubbo/douyu", "root path for test keys to be written")
	cmd.Flags().StringVar(&zookeeperServers, "zookeeper-servers", "", "comma-separated list of zookeeper servers address")
	cmd.Flags().UintVar(&concurrent, "concurrent", 50, "the concurreny of syncing worker")
	return cmd
}
