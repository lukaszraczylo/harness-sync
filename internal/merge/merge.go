// Package merge wraps git merge-file for three-way file merges.
package merge

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Inputs:
//
//	Base   = last-rendered snapshot (state/...)
//	Ours   = new rendered output (what we'd write now)
//	Theirs = current target file contents (what's on disk at the destination)
type Inputs struct {
	Base   []byte
	Ours   []byte
	Theirs []byte
}

type Result struct {
	Body     []byte
	Conflict bool
}

// ThreeWay runs `git merge-file -p ours base theirs` and returns the
// merged body. When Conflict is true, Body contains conflict markers.
func ThreeWay(in Inputs) (*Result, error) {
	tmp, err := os.MkdirTemp("", "harness-sync-merge-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	oursPath := filepath.Join(tmp, "ours")
	basePath := filepath.Join(tmp, "base")
	theirsPath := filepath.Join(tmp, "theirs")

	if err := os.WriteFile(oursPath, in.Ours, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(basePath, in.Base, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(theirsPath, in.Theirs, 0o644); err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"git", "merge-file",
		"-L", "harness-sync (new)",
		"-L", "harness-sync (base)",
		"-L", "harness-sync (current)",
		"-p",
		oursPath, basePath, theirsPath,
	)
	out, runErr := cmd.Output()

	res := &Result{Body: out}
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			// git merge-file exits with conflict-count on conflict; body still valid
			if ee.ExitCode() > 0 {
				res.Conflict = true
				return res, nil
			}
		}
		return nil, fmt.Errorf("git merge-file: %w", runErr)
	}
	return res, nil
}
