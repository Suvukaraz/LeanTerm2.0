// LeanTerm patch: additions to Gio's app package used by LeanTerm. See
// third_party/gio/LEANTERM.md for the list of changes to the upstream code.

package app

import (
	"image"

	"gioui.org/f32"
	"gioui.org/io/system"
	"gioui.org/unit"
)

// FileDropEvent is sent when files are dropped on the window from another
// application, such as a file manager. It is delivered by Window.Event on
// X11 (XDND) and Wayland (wl_data_device); on other platforms it is not sent.
type FileDropEvent struct {
	// Paths are the local paths of the dropped files.
	Paths []string
	// Position is where they were dropped, in pixels relative to the
	// window's content (the coordinates of FrameEvent).
	Position f32.Point
}

func (FileDropEvent) ImplementsEvent() {}

// AppID sets the application id of this window, overriding ID: the Wayland
// app_id and the X11 WM_CLASS that desktops use to find the window's
// .desktop file, taskbar group and icon. It applies when the window is
// created.
func AppID(id string) Option {
	return func(_ unit.Metric, cnf *Config) {
		cnf.appID = id
	}
}

// Icon sets the window's icon where the window system takes one per window
// (X11 _NET_WM_ICON). Wayland desktops take the icon of the .desktop file
// that matches the app id; Windows takes the executable's icon. It applies
// when the window is created.
func Icon(img image.Image) Option {
	return func(_ unit.Metric, cnf *Config) {
		cnf.icon = img
	}
}

// activator is implemented by window drivers that can activate a window
// with a token of the desktop.
type activator interface {
	activate(token string)
}

// Activate raises the window and asks for the keyboard focus. token is an
// activation token from the desktop, such as XDG_ACTIVATION_TOKEN (Wayland)
// or DESKTOP_STARTUP_ID (X11) of a process that was just started by the
// user and hands its work over to this window; without one, the window asks
// for activation itself, which the desktop may turn into a request for
// attention. On platforms without tokens it is Perform(system.ActionRaise).
func (w *Window) Activate(token string) {
	if w.driver == nil {
		w.Perform(system.ActionRaise)
		return
	}
	w.Run(func() {
		if w.driver == nil {
			return
		}
		if a, ok := w.driver.(activator); ok {
			a.activate(token)
			return
		}
		w.driver.Perform(system.ActionRaise)
	})
}
