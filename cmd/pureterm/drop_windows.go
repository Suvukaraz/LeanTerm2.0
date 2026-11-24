//go:build windows

package main

import (
	"log"

	"gioui.org/app"

	"github.com/Suvukaraz/PureTerm/internal/gui/dnd"
)

// maybeEnableDrop makes the main window accept dropped files (see dnd).
func (u *ui) maybeEnableDrop(e app.ViewEvent) {
	enableWindowsDrop(e, &u.dropHWND, u.onDrop)
}

// enableWindowsDrop subclasses a window to accept dropped files; last is the
// HWND it was done for.
//
// Gio's window thread waits while the program handles an event, and
// subclassing sends the window messages: called here directly, both wait
// for each other (the window hung at its start). It runs in its own
// goroutine, as in the other drop handlers.
func enableWindowsDrop(e app.ViewEvent, last *uintptr, onDrop func(dnd.Drop)) {
	ev, ok := e.(app.Win32ViewEvent)
	if !ok || !ev.Valid() || ev.HWND == *last {
		return
	}
	*last = ev.HWND
	h := uintptr(ev.HWND)
	go func() {
		if err := dnd.Enable(h, onDrop); err != nil {
			log.Print("file drop disabled: ", err)
			return
		}
		log.Print("file drop enabled")
	}()
}
