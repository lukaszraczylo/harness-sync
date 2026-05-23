// Package gitx wraps git CLI invocations the canonical repo needs.
package gitx

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Repo struct {
	Dir string
}

func New(dir string) *Repo { return &Repo{Dir: dir} }

func (r *Repo) run(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %v: %w: %s", args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (r *Repo) Init() error {
	_, err := r.run("init", "-q", "-b", "main")
	return err
}

func (r *Repo) Configure(name, email string) error {
	if _, err := r.run("config", "user.name", name); err != nil {
		return err
	}
	_, err := r.run("config", "user.email", email)
	return err
}

func (r *Repo) AddAll() error {
	_, err := r.run("add", "-A")
	return err
}

func (r *Repo) Add(paths ...string) error {
	args := append([]string{"add", "--"}, paths...)
	_, err := r.run(args...)
	return err
}

func (r *Repo) Commit(msg string) error {
	// --no-verify: harness-sync makes internal state-tracking commits in the
	// canonical repo; user-configured hooks (secret scanners, linters) must not
	// block them — those hooks belong to the harness-sync source repo, not here.
	_, err := r.run("commit", "-q", "--no-verify", "-m", msg)
	return err
}

func (r *Repo) HeadCommit() (string, error) {
	b, err := r.run("rev-parse", "HEAD")
	return string(bytes.TrimSpace(b)), err
}

func (r *Repo) ShowFileAtHead(path string) ([]byte, error) {
	return r.run("show", "HEAD:"+path)
}

func (r *Repo) IsRepo() bool {
	_, err := r.run("rev-parse", "--git-dir")
	return err == nil
}

// Revert undoes the last N commits with `git revert --no-edit HEAD~N..HEAD`.
// n must be >= 1.
func (r *Repo) Revert(n int) error {
	if n < 1 {
		return fmt.Errorf("revert: n must be >= 1, got %d", n)
	}
	_, err := r.run("revert", "--no-edit", fmt.Sprintf("HEAD~%d..HEAD", n))
	return err
}
