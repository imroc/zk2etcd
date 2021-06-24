package main

import (
	"fmt"
	"github.com/imroc/zk2etcd/pkg/version"
	"github.com/spf13/cobra"
)

const versionDesc = `
version print the version info for cls
`

func newVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "print version",
		Long:  versionDesc,
		Run: func(cmd *cobra.Command, args []string) {
			v := version.GetVersion()
			fmt.Printf("%+#v\n", v)
		},
	}
	return cmd
}
