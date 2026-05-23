package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

const installURL = "https://raw.githubusercontent.com/lukaszraczylo/harness-sync/main/install.sh"

// NewUpdate returns the `update` subcommand. It downloads and installs the
// latest harness-sync release into the same directory the running binary
// lives in.
func NewUpdate() *cobra.Command {
	var (
		dryRun bool
		dir    string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Download and install the latest harness-sync release in place",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dir == "" {
				exe, err := os.Executable()
				if err != nil {
					return fmt.Errorf("resolve current binary path: %w", err)
				}
				resolved, err := filepath.EvalSymlinks(exe)
				if err == nil {
					exe = resolved
				}
				dir = filepath.Dir(exe)
			}
			pipeline := fmt.Sprintf("curl -fsSL %q | INSTALL_DIR=%q bash", installURL, dir)
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Installing latest harness-sync into %s\n", dir); err != nil {
				return err
			}
			if dryRun {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), pipeline)
				return err
			}
			sh := exec.Command("/bin/bash", "-c", pipeline)
			sh.Stdout = cmd.OutOrStdout()
			sh.Stderr = cmd.ErrOrStderr()
			return sh.Run()
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the install command without executing it")
	cmd.Flags().StringVar(&dir, "install-dir", "", "override install directory (default: dir of running binary)")
	return cmd
}
