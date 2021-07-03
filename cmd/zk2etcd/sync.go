package main

import (
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/sync"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func newSyncCmd(args []string) *cobra.Command {
	var common Common
	stopChan := make(chan struct{}, 1)
	var concurrent uint
	var fullSyncInterval time.Duration
	var logOption log.Options
	var zkOption zookeeper.Options
	var etcdOption etcd.Options

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "sync zookeeper to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		PreRun: func(cmd *cobra.Command, args []string) {
			log.Init(&logOption) // 初始化 logging
			zookeeper.Init(&zkOption)
			etcd.Init(&etcdOption)

			// TODO: 初始化 zk 和 etcd
		},
		Run: func(cmd *cobra.Command, args []string) {

			zkPrefixes, zkExcludePrefixes := common.GetAll()

			s := sync.New(zkPrefixes, zkExcludePrefixes, concurrent, fullSyncInterval, stopChan)
			go s.Run()

			// TODO: 实现真正优雅停止
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			stopChan <- struct{}{}
		},
	}

	cmd.SetArgs(args)
	common.AddFlags(cmd.Flags())
	logOption.AddFlags(cmd.Flags())
	zkOption.AddFlags(cmd.Flags())
	etcdOption.AddFlags(cmd.Flags())

	cmd.Flags().UintVar(&concurrent, "concurrent", 50, "the concurreny of syncing worker")
	cmd.Flags().DurationVar(&fullSyncInterval, "fullsync-interval", 5*time.Minute, "the interval of full sync, set to 0s to disable it")
	return cmd
}
