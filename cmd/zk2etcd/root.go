package main

import (
	"github.com/spf13/cobra"
)

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "zk2etcd",
		Short: "zookeeper sync to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
	}

	rootCmd.SetArgs(args)
	rootCmd.AddCommand(newSyncCmd(args))
	rootCmd.AddCommand(newDiffCmd(args))
	rootCmd.AddCommand(newGenzkCmd(args))
	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}
