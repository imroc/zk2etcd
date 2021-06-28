package main

import (
	"github.com/imroc/zk2etcd/pkg/sync"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

func newSyncCmd(args []string) *cobra.Command {
	var common Common
	stopChan := make(chan struct{}, 1)
	var concurrent uint

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "sync zookeeper to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		Run: func(cmd *cobra.Command, args []string) {
			zkClient, etcdClient, logger, zkPrefixes, zkExcludePrefixes := common.GetAll()

			s := sync.New(zkClient, zkPrefixes, zkExcludePrefixes, etcdClient, logger, concurrent)
			go s.Run(stopChan)

			// TODO: 实现真正优雅停止
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			stopChan <- struct{}{}
		},
	}

	cmd.SetArgs(args)
	common.AddFlags(cmd.Flags())
	cmd.Flags().UintVar(&concurrent, "concurrent", 20, "the concurreny of syncing worker")
	return cmd
}
