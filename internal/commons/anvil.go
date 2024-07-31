//go:build !windows

package commons

import "context"

// Prerequisites implements HandleRelease.
func (a *AnvilRelease) Prerequisites(ctx context.Context) error {
	return nil
}
