//go:build !windows

package winsys

import "gioui.org/app"

func setProcessAppID(*App)     {}
func allowActivation()         {}
func view(*App, app.ViewEvent) {}

// SetShortcutAppID is only needed on Windows.
func SetShortcutAppID(string, string) error { return nil }
