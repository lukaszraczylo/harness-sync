// Package apply orchestrates adapter rendering, three-way merge, and writes.
package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lukaszraczylo/harness-sync/internal/adapter"
	"github.com/lukaszraczylo/harness-sync/internal/canonical"
	"github.com/lukaszraczylo/harness-sync/internal/gitx"
	"github.com/lukaszraczylo/harness-sync/internal/merge"
)

// Options configures a Run call.
type Options struct {
	Bundle   *canonical.Bundle
	Repo     *gitx.Repo
	Adapters []adapter.Adapter
	DryRun   bool
	Force    bool
}

// Report summarises what Run did.
type Report struct {
	Actions   []Action
	Written   int
	Skipped   int
	Conflicts int
}

// Action describes one file's outcome.
type Action struct {
	Adapter string
	Dest    string
	Kind    string // "wrote" | "symlinked" | "skipped" | "conflict"
	Note    string
}

// Run renders all adapters and writes their FileSets to disk.
// Conflicts are recorded as <dest>.rej files; processing continues past conflicts.
//
// IMPORTANT: harness-sync intentionally does NOT resolve ${VAR} references in
// canonical configs. Every supported harness performs its own env-var
// expansion at MCP launch / config-load time, so resolving here would only
// bake real secret values into the rendered target files AND the state/
// snapshots tracked by git — which is a security regression. Leave
// placeholders alone; the downstream harness resolves them at use time.
func Run(opt Options) (*Report, error) {
	rep := &Report{}
	for _, ad := range opt.Adapters {
		files, err := ad.Render(opt.Bundle)
		if err != nil {
			return rep, fmt.Errorf("render %s: %w", ad.Name(), err)
		}
		var renderErr error
		files.ForEach(func(f adapter.File) {
			if renderErr != nil {
				return
			}
			renderErr = handle(opt, ad.Name(), f, rep)
		})
		if renderErr != nil {
			return rep, renderErr
		}
	}
	if !opt.DryRun && rep.Written > 0 && opt.Repo != nil {
		if err := opt.Repo.AddAll(); err != nil {
			return rep, err
		}
		if err := opt.Repo.Commit(fmt.Sprintf("apply: %d files, %d conflicts", rep.Written, rep.Conflicts)); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

func handle(opt Options, adapterName string, f adapter.File, rep *Report) error {
	switch f.Kind {
	case adapter.RenderedFile:
		return handleRendered(opt, adapterName, f, rep)
	case adapter.SymlinkFile, adapter.SymlinkDir:
		return handleSymlink(opt, adapterName, f, rep)
	}
	return fmt.Errorf("unknown file kind for %s", f.Dest)
}

func statePath(root, adapterName, dest string) string {
	return filepath.Join(root, "state", adapterName, dest)
}

func handleRendered(opt Options, adapterName string, f adapter.File, rep *Report) error {
	sp := statePath(opt.Bundle.Root, adapterName, f.Dest)
	base, _ := os.ReadFile(sp)
	current, _ := os.ReadFile(f.Dest)

	if string(current) == string(f.Content) {
		rep.Skipped++
		rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "skipped", Note: "already in sync"})
		return writeState(opt, sp, f.Content)
	}

	if opt.Force || len(base) == 0 || string(current) == string(base) {
		if opt.DryRun {
			rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote", Note: "would write"})
			return nil
		}
		if err := writeFile(f.Dest, f.Content, f.Mode); err != nil {
			return err
		}
		if err := writeState(opt, sp, f.Content); err != nil {
			return err
		}
		rep.Written++
		rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote"})
		return nil
	}

	res, err := merge.ThreeWay(merge.Inputs{Base: base, Ours: f.Content, Theirs: current})
	if err != nil {
		return err
	}
	if res.Conflict {
		if opt.DryRun {
			rep.Conflicts++
			rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "conflict", Note: "would write .rej"})
			return nil
		}
		if err := os.WriteFile(f.Dest+".rej", res.Body, 0o600); err != nil {
			return err
		}
		rep.Conflicts++
		rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "conflict", Note: "wrote .rej"})
		return nil
	}
	if opt.DryRun {
		rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote", Note: "would merge"})
		return nil
	}
	if err := writeFile(f.Dest, res.Body, f.Mode); err != nil {
		return err
	}
	if err := writeState(opt, sp, f.Content); err != nil {
		return err
	}
	rep.Written++
	rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "wrote", Note: "merged"})
	return nil
}

func handleSymlink(opt Options, adapterName string, f adapter.File, rep *Report) error {
	existing, err := os.Readlink(f.Dest)
	if err == nil && existing == f.SymlinkTarget {
		rep.Skipped++
		rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "skipped", Note: "symlink already correct"})
		return nil
	}
	if opt.DryRun {
		rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "symlinked", Note: "would link"})
		return nil
	}
	if _, statErr := os.Lstat(f.Dest); statErr == nil {
		backup := filepath.Join(opt.Bundle.Root, "backups", adapterName, filepath.Base(f.Dest))
		if err := os.MkdirAll(filepath.Dir(backup), 0o750); err != nil {
			return err
		}
		if renameErr := os.Rename(f.Dest, backup); renameErr != nil && !errors.Is(renameErr, fs.ErrNotExist) {
			return renameErr
		}
	}
	if err := os.MkdirAll(filepath.Dir(f.Dest), 0o750); err != nil {
		return err
	}
	if err := os.Symlink(f.SymlinkTarget, f.Dest); err != nil {
		return err
	}
	rep.Written++
	rep.Actions = append(rep.Actions, Action{Adapter: adapterName, Dest: f.Dest, Kind: "symlinked"})
	return nil
}

func writeFile(dest string, body []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if mode == 0 {
		mode = 0o644
	}
	if err := os.WriteFile(tmp, body, mode); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

func writeState(opt Options, sp string, body []byte) error {
	if opt.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(sp), 0o750); err != nil {
		return err
	}
	return os.WriteFile(sp, body, 0o600)
}
