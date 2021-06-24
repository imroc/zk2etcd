package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/controller"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	var zkAddr, zkPrefix, etcdAddr string
	stopChan := make(chan struct{}, 1)
	rootCmd := &cobra.Command{
		Use:   "zk2etcd",
		Short: "zookeeper sync to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		Run: func(cmd *cobra.Command, args []string) {
			c := controller.New(zkAddr, zkPrefix, etcdAddr)
			go c.Run(stopChan)
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			stopChan <- struct{}{}
		},
	}
	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVar(&zkAddr, "zkAddr", "", "zookeeper address")
	rootCmd.PersistentFlags().StringVar(&zkPrefix, "zkPrefix", "/dubbo", "zookeeper prefix")
	rootCmd.PersistentFlags().StringVar(&etcdAddr, "etcdAddr", "", "zookeeper address")
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
