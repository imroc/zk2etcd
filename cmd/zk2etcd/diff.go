package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/diff"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/record"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"github.com/spf13/cobra"
	"time"
)

func newDiffCmd(args []string) *cobra.Command {
	var common Common
	var concurrent uint
	var fix bool
	var logOption log.Options
	var zkOption zookeeper.Options
	var etcdOption etcd.Options
	var recordOption record.Options

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "compare keys between zk and etcd",

		DisableAutoGenTag: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			log.Init(&logOption) // 初始化 logging
			zookeeper.Init(&zkOption)
			etcd.Init(&etcdOption)
			record.Init(&recordOption)
		},
		Run: func(cmd *cobra.Command, args []string) {
			zkPrefixes, zkExcludePrefixes := common.GetAll()
			d := diff.New(zkPrefixes, zkExcludePrefixes, concurrent)
			before := time.Now()
			d.Run()
			if fix {
				missedCount, extraCount := d.Fix()
				fmt.Println("put missed count: ", missedCount)
				fmt.Println("delete extra count: ", extraCount)
				d.Recheck()
			}
			d.PrintSummary()
			cost := time.Since(before)
			fmt.Println("total cost: ", cost)
		},
	}

	cmd.SetArgs(args)
	common.AddFlags(cmd.Flags())
	logOption.AddFlags(cmd.Flags())
	zkOption.AddFlags(cmd.Flags())
	etcdOption.AddFlags(cmd.Flags())
	recordOption.AddFlags(cmd.Flags())

	cmd.Flags().UintVar(&concurrent, "concurrent", 50, "the concurreny of syncing worker")
	cmd.Flags().BoolVar(&fix, "fix", false, "set to true will fix the data diff between zk and etcd")
	return cmd
}
