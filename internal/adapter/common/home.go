package common

import "os"

// DefaultHome returns the current user's home directory, or "" on error.
func DefaultHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}
