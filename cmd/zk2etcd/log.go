package main

import (
	"fmt"
	"github.com/imroc/req"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func newLogCmd(args []string) *cobra.Command {
	var level, addr string
	cmd := &cobra.Command{
		Use:               "log",
		Short:             "get and set log settings",
		DisableAutoGenTag: true,
		Run: func(cmd *cobra.Command, args []string) {
			if addr == "" {
				fmt.Fprintln(os.Stderr, "need specify the zk2etcd http address")
				return
			}
			if level != "" {
				addr = strings.TrimSuffix(addr, "/")
				api := addr + "/log"
				resp, err := req.Post(api, req.Param{
					"level": level,
				})
				if err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
					return
				}
				fmt.Println(resp.String())
			}
		},
	}
	cmd.Flags().StringVar(&level, "level", "", "set log level")
	cmd.Flags().StringVar(&addr, "zk2etcd-addr", "http://localhost:80", "zk2etcd http address")
	cmd.SetArgs(args)
	return cmd
}
