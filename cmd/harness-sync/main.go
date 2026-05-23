package main

import (
	"fmt"
	"os"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/cagent"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/claudecode"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/crush"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/goose"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/kilo"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/opencode"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/zed"
	"github.com/lukaszraczylo/harness-sync/internal/cli"
)

var version = "dev"

func main() {
	reg := adapter.NewRegistry()
	reg.Register(claudecode.New())
	reg.Register(crush.New())
	reg.Register(kilo.New())
	reg.Register(opencode.New())
	reg.Register(goose.New())
	reg.Register(cagent.New())
	reg.Register(zed.New())

	root := cli.NewRoot(version)
	root.AddCommand(cli.NewDetect(reg))
	root.AddCommand(cli.NewApply(reg))
	root.AddCommand(cli.NewDiff(reg))
	root.AddCommand(cli.NewShow(reg))
	root.AddCommand(cli.NewInit(reg))
	root.AddCommand(cli.NewProfile(reg))
	root.AddCommand(cli.NewRollback())
	root.AddCommand(cli.NewAdapter(reg))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
