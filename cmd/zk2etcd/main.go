package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/controller"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	var zkAddr, zkPrefix, etcdAddr, logLevel string
	stopChan := make(chan struct{}, 1)
	rootCmd := &cobra.Command{
		Use:   "zk2etcd",
		Short: "zookeeper sync to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.New(logLevel)
			zkServers := strings.Split(zkAddr, ",")
			zkClient := zookeeper.NewClient(logger, zkServers)
			etcdServers := strings.Split(etcdAddr, ",")
			etcdClient := etcd.NewClient(logger, etcdServers)
			c := controller.New(zkClient, zkPrefix, etcdClient, logger)
			go c.Run(stopChan)
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			stopChan <- struct{}{}
		},
	}
	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVar(&zkAddr, "zookeeper-servers", "", "comma-separated list of zookeeper servers address")
	rootCmd.PersistentFlags().StringVar(&zkPrefix, "zookeeper-prefix", "/dubbo", "the zookeeper path prefix to be synced")
	rootCmd.PersistentFlags().StringVar(&etcdAddr, "etcd-servers", "", "comma-separated list of etcd servers address")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log output level，possible values: 'debug', 'info', 'warn', 'error', 'panic', 'fatal'")

	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

func main() {
	rootCmd := GetRootCmd(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		//exitCode := cmd.GetExitCode(err)
		//os.Exit(exitCode)
		fmt.Println(err)
	}
}
