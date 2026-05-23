package main

import (
	"fmt"
	"os"

	"github.com/lukaszraczylo/harness-sync/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRoot(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
