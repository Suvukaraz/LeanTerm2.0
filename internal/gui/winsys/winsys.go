// Package winsys gives LeanTerm's and LeanEdit's windows their identity in
// the desktop: the application id (Wayland app_id, X11 WM_CLASS, Windows
// AppUserModelID) that taskbars group windows by and that names the .desktop
// file with the icon, the window icon, and the activation of a window on
// behalf of a process the user just started.
//
// LeanEdit is an application of its own in the taskbar, also when its window
// belongs to the LeanTerm process (opened from LeanTerm's file browser).
package winsys

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"os"
	"sync"

	"gioui.org/app"
)

//go:embed icons/leanterm.png
var leantermPNG []byte

//go:embed icons/leanedit.png
var leaneditPNG []byte

// App is the identity of one of the applications.
type App struct {
	// ID is the application id: the name of the .desktop file without
	// ".desktop", its StartupWMClass, the Wayland app_id, the X11 WM_CLASS
	// and the Windows AppUserModelID.
	ID string
	// Name is the name shown to the user.
	Name string
	// Exe is the name of the application's executable on Windows
	// (leanedit.exe runs LeanEdit; leanterm.exe -leanedit does as well).
	Exe string
	// IconResource is the icon's resource id in both Windows executables.
	IconResource int

	png  []byte
	once sync.Once
	icon image.Image
}

var (
	// LeanTerm is the connection manager's identity.
	LeanTerm = &App{ID: "org.leanterm.LeanTerm", Name: "LeanTerm", Exe: "leanterm.exe", IconResource: 1, png: leantermPNG}
	// LeanEdit is the text editor's identity.
	LeanEdit = &App{ID: "org.leanterm.LeanEdit", Name: "LeanEdit", Exe: "leanedit.exe", IconResource: 2, png: leaneditPNG}
)

// Icon is the application's icon (256x256).
func (a *App) Icon() image.Image {
	a.once.Do(func() {
		img, err := png.Decode(bytes.NewReader(a.png))
		if err == nil {
			a.icon = img
		}
	})
	return a.icon
}

// Options are the window options of a window of the application: its id
// and icon.
func (a *App) Options() []app.Option {
	opts := []app.Option{app.AppID(a.ID)}
	if icon := a.Icon(); icon != nil {
		opts = append(opts, app.Icon(icon))
	}
	return opts
}

// SetProcess makes a the application of this process: the id of windows
// without their own (Gio's app.ID) and, on Windows, the process's
// AppUserModelID. It must run before the first window is created.
func SetProcess(a *App) {
	app.ID = a.ID
	setProcessAppID(a)
}

var (
	tokenOnce sync.Once
	token     string
)

// StartupToken returns the activation token the desktop gave this process
// when the user started it (Wayland: XDG_ACTIVATION_TOKEN, X11:
// DESKTOP_STARTUP_ID), once: a token is used for one activation. It removes
// the token from the environment, so programs started later (shells, New
// Window) do not inherit a used one.
func StartupToken() string {
	tokenOnce.Do(func() {
		for _, name := range []string{"XDG_ACTIVATION_TOKEN", "DESKTOP_STARTUP_ID"} {
			if v := os.Getenv(name); v != "" && token == "" {
				token = v
			}
			os.Unsetenv(name)
		}
	})
	t := token
	token = ""
	return t
}

// AllowActivation lets the running instance this process hands its work to
// take the foreground (Windows; a no-op elsewhere, where the token does it).
func AllowActivation() { allowActivation() }

// View integrates a window when its native view exists (call it with every
// app.ViewEvent): on Windows the taskbar identity and icon of a LeanEdit
// window in a LeanTerm process. It returns at once; the work runs in the
// background, as calls into the window's thread must not wait for it while
// it waits for the event's handling.
func View(a *App, e app.ViewEvent) { view(a, e) }
