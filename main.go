package main

import (
	"fmt"
	"os"

	"github.com/kubeone/go-time-tracker/cmd"
)

var version = "dev"

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
