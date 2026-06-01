package adapter

import "github.com/lukaszraczylo/harness-sync/internal/adapter/common"

// Base holds the cross-adapter defaults (currently just the home directory
// used to compute target paths). Embed it in each adapter struct to inherit
// the WithHome/New constructor pattern without re-implementing it.
type Base struct {
	Home string
}

// BaseOption configures a Base. Returned by WithHome and consumed by NewBase.
type BaseOption func(*Base)

// WithHome overrides the home directory used for target paths. Defaults to
// common.DefaultHome() (current $HOME) when no option is supplied.
func WithHome(h string) BaseOption {
	return func(b *Base) { b.Home = h }
}

// NewBase returns a Base populated from opts, with Home defaulting to the
// current user's home directory.
func NewBase(opts ...BaseOption) *Base {
	b := &Base{Home: common.DefaultHome()}
	for _, o := range opts {
		o(b)
	}
	return b
}
