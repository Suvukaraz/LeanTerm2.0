//go:build !windows

package main

import (
	"gioui.org/app"

	"github.com/Suvukaraz/PureTerm/internal/gui/dnd"
)

// maybeEnableDrop does nothing on X11 and Wayland: the patched Gio delivers
// app.FileDropEvent (see third_party/gio).
func (u *ui) maybeEnableDrop(e app.ViewEvent) {}

func enableWindowsDrop(app.ViewEvent, *uintptr, func(dnd.Drop)) {}
