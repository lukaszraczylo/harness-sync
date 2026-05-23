// Package ui provides small interactive prompts and their non-interactive
// equivalents for tests.
package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

type msOpts struct {
	nonInteractive []string
	interactive    bool
}

// Option configures MultiSelect behaviour.
type Option func(*msOpts)

// WithNonInteractive bypasses the TUI entirely and returns the given choices.
// Test code uses this to make MultiSelect deterministic.
func WithNonInteractive(choices []string) Option {
	return func(o *msOpts) {
		o.nonInteractive = choices
		o.interactive = false
	}
}

// MultiSelect prompts the user (or returns the pre-set non-interactive value).
// Choices passed via WithNonInteractive must all appear in `choices`.
func MultiSelect(title string, choices []string, opts ...Option) ([]string, error) {
	o := &msOpts{interactive: true}
	for _, fn := range opts {
		fn(o)
	}
	if !o.interactive {
		valid := map[string]bool{}
		for _, c := range choices {
			valid[c] = true
		}
		for _, c := range o.nonInteractive {
			if !valid[c] {
				return nil, fmt.Errorf("choice %q not in %v", c, choices)
			}
		}
		return o.nonInteractive, nil
	}
	// Pre-select every choice. Users deselect what they don't want; hitting
	// enter without toggling thus accepts all (the common case) instead of
	// silently producing an empty result.
	picked := make([]string, len(choices))
	copy(picked, choices)

	opts2 := make([]huh.Option[string], 0, len(choices))
	for _, c := range choices {
		opts2 = append(opts2, huh.NewOption(c, c).Selected(true))
	}
	form := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Description("space to toggle · enter to confirm · all pre-selected").
			Options(opts2...).
			Value(&picked),
	))
	if err := form.Run(); err != nil {
		return nil, err
	}
	if len(picked) == 0 {
		return nil, fmt.Errorf("no items selected")
	}
	return picked, nil
}
