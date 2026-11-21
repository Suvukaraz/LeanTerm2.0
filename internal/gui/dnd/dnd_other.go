//go:build !windows

package dnd

// Enable is not needed on X11 and Wayland: Gio (third_party/gio) handles
// XDND and Wayland drops and delivers app.FileDropEvent.
func Enable(uintptr, func(Drop)) error { return ErrUnsupported }

// SimulateDrop is only implemented on Windows.
func SimulateDrop(uintptr, []string, int, int) error { return ErrUnsupported }
