package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/diff"
	"github.com/spf13/cobra"
	"time"
)

func newDiffCmd(args []string) *cobra.Command {
	var common Common
	var concurrent uint
	var fix bool

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "compare keys between zk and etcd",

		DisableAutoGenTag: true,

		Run: func(cmd *cobra.Command, args []string) {
			zkClient, etcdClient, logger, zkPrefixes, zkExcludePrefixes := common.GetAll()
			d := diff.New(zkClient, zkPrefixes, zkExcludePrefixes, etcdClient, logger, concurrent)
			before := time.Now()
			d.Run()
			d.PrintSummary()
			if fix {
				d.Fix()
			}
			cost := time.Since(before)
			fmt.Println("total cost: ", cost)
		},
	}

	cmd.SetArgs(args)
	common.AddFlags(cmd.Flags())
	cmd.Flags().UintVar(&concurrent, "concurrent", 50, "the concurreny of syncing worker")
	cmd.Flags().BoolVar(&fix, "fix", false, "set to true will fix the data diff between zk and etcd")
	return cmd
}
