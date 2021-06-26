package main

import (
	"fmt"
	"os"
)

func main() {
	rootCmd := GetRootCmd(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
