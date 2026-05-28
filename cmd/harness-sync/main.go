package main

import (
	"fmt"
	"os"
	"time"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/cagent"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/claudecode"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/crush"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/goose"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/kilo"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/opencode"
	"github.com/lukaszraczylo/harness-sync/internal/adapters/zed"
	"github.com/lukaszraczylo/harness-sync/internal/cli"
	"github.com/lukaszraczylo/oss-telemetry"
)

var version = "dev"

func main() {
	telemetry.SendForModule("harness-sync", "github.com/lukaszraczylo/harness-sync", version)
	defer telemetry.Wait(2 * time.Second)

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
	root.AddCommand(cli.NewRollback(reg))
	root.AddCommand(cli.NewAdapter(reg))
	root.AddCommand(cli.NewUpdate())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
