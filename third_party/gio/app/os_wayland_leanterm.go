// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd) && !nowayland

// LeanTerm patch: files dropped on a window (wl_data_device drag and drop,
// delivered as FileDropEvent) and window activation (xdg-activation-v1).

package app

//go:generate wayland-scanner client-header /usr/share/wayland-protocols/staging/xdg-activation/xdg-activation-v1.xml wayland_xdg_activation.h
//go:generate wayland-scanner private-code /usr/share/wayland-protocols/staging/xdg-activation/xdg-activation-v1.xml wayland_xdg_activation.c
//go:generate sed -i "1s;^;//go:build ((linux \\&\\& !android) || freebsd) \\&\\& !nowayland\\n\\n;" wayland_xdg_activation.c

/*
#cgo linux pkg-config: wayland-client

#include <stdlib.h>
#include <wayland-client.h>
#include "wayland_xdg_activation.h"

extern const struct xdg_activation_token_v1_listener gio_xdg_activation_token_v1_listener;
*/
import "C"

import (
	"io"
	"os"
	"slices"
	"unsafe"

	"gioui.org/f32"
)

// wlLeanTerm is the display's state of the LeanTerm patch.
type wlLeanTerm struct {
	activation *C.struct_xdg_activation_v1
}

// wlDnD is a drag over a window of the seat's display.
type wlDnD struct {
	offer    *C.struct_wl_data_offer
	win      *window
	pos      f32.Point
	accepted bool
	// dropping: the drop happened and the data is being read; the offer
	// lives until finishDrop.
	dropping bool
}

// wlDrop is a finished read of dropped data, handed to the window's loop.
type wlDrop struct {
	offer *C.struct_wl_data_offer
	event FileDropEvent
}

func (d *wlDisplay) bindActivation(reg *C.struct_wl_registry, name C.uint32_t) {
	d.lt.activation = (*C.struct_xdg_activation_v1)(C.wl_registry_bind(reg, name, &C.xdg_activation_v1_interface, 1))
}

func (d *wlDisplay) destroyLeanTerm() {
	if d.lt.activation != nil {
		C.xdg_activation_v1_destroy(d.lt.activation)
		d.lt.activation = nil
	}
}

func dndPos(w *window, x, y C.wl_fixed_t) f32.Point {
	scale := float32(1)
	if w != nil {
		scale = float32(w.scale)
	}
	return f32.Point{X: fromFixed(x) * scale, Y: fromFixed(y) * scale}
}

// dragMove reports a drag position to the window (nil: none of ours).
func (w *window) dragMove(pos f32.Point) {
	if w != nil {
		dragMove(w.ProcessEvent, pos)
	}
}

// dropOffer forgets and destroys an offer.
func (s *wlSeat) dropOffer(o *C.struct_wl_data_offer) {
	if o == nil || o == s.clipboard {
		return
	}
	delete(s.offers, o)
	callbackDelete(unsafe.Pointer(o))
	C.wl_data_offer_destroy(o)
}

// dndEnter starts a drag over one of the display's surfaces: offers with
// files (text/uri-list) are accepted for copying.
func (s *wlSeat) dndEnter(serial C.uint32_t, surf *C.struct_wl_surface, x, y C.wl_fixed_t, id *C.struct_wl_data_offer) {
	if !s.dnd.dropping && s.dnd.offer != nil && s.dnd.offer != id {
		s.dropOffer(s.dnd.offer)
	}
	if !s.dnd.dropping {
		s.dnd = wlDnD{}
	}
	if id == nil || s.dnd.dropping {
		return
	}
	var w *window
	if surf != nil {
		w, _ = callbackLoad(unsafe.Pointer(surf)).(*window)
	}
	s.dnd = wlDnD{offer: id, win: w, pos: dndPos(w, x, y)}
	w.dragMove(s.dnd.pos)
	if w != nil && slices.Contains(s.offers[id], uriListMime) {
		mime := C.CString(uriListMime)
		defer C.free(unsafe.Pointer(mime))
		C.wl_data_offer_accept(id, serial, mime)
		C.wl_data_offer_set_actions(id, C.WL_DATA_DEVICE_MANAGER_DND_ACTION_COPY, C.WL_DATA_DEVICE_MANAGER_DND_ACTION_COPY)
		s.dnd.accepted = true
		return
	}
	C.wl_data_offer_accept(id, serial, nil)
}

func (s *wlSeat) dndMotion(x, y C.wl_fixed_t) {
	if s.dnd.offer != nil && !s.dnd.dropping {
		s.dnd.pos = dndPos(s.dnd.win, x, y)
		s.dnd.win.dragMove(s.dnd.pos)
	}
}

func (s *wlSeat) dndLeave() {
	if s.dnd.dropping {
		return
	}
	if s.dnd.offer != nil {
		s.dnd.win.dragMove(dragOutside)
	}
	s.dropOffer(s.dnd.offer)
	s.dnd = wlDnD{}
}

// dndDrop reads the dropped file list in the background; the window's loop
// finishes the drop (finishDrop) when it has arrived.
func (s *wlSeat) dndDrop() {
	dnd := s.dnd
	if dnd.offer == nil || !dnd.accepted || dnd.win == nil || dnd.dropping {
		s.dndLeave()
		return
	}
	r, wr, err := os.Pipe()
	if err != nil {
		s.dndLeave()
		return
	}
	mime := C.CString(uriListMime)
	C.wl_data_offer_receive(dnd.offer, mime, C.int(wr.Fd()))
	C.free(unsafe.Pointer(mime))
	// wl_data_offer_receive duplicates the descriptor when the request is
	// sent: flush before closing ours.
	C.wl_display_flush(s.disp.disp)
	wr.Close()
	s.dnd.dropping = true
	w := dnd.win
	go func() {
		defer r.Close()
		data, _ := io.ReadAll(io.LimitReader(r, 16<<20))
		w.ltDrops <- wlDrop{offer: dnd.offer, event: FileDropEvent{Paths: parseURIList(string(data)), Position: dnd.pos}}
		w.disp.wakeup()
	}()
}

// finishDrop ends a drop on the window's loop and delivers its files.
func (w *window) finishDrop(d wlDrop) {
	if s := w.disp.seat; s != nil {
		C.wl_data_offer_finish(d.offer)
		if s.dnd.offer == d.offer {
			s.dnd = wlDnD{}
		}
		s.dropOffer(d.offer)
	}
	if len(d.event.Paths) > 0 {
		w.ProcessEvent(d.event)
	}
}

// activate raises the window with xdg-activation: with the token of the
// process that handed its work over, else with a token the window asks for
// itself (with its latest input serial, if any).
func (w *window) activate(token string) {
	act := w.disp.lt.activation
	if act == nil || w.surf == nil {
		return
	}
	if token != "" {
		t := C.CString(token)
		C.xdg_activation_v1_activate(act, t, w.surf)
		C.free(unsafe.Pointer(t))
		C.wl_display_flush(w.disp.disp)
		return
	}
	tok := C.xdg_activation_v1_get_activation_token(act)
	if tok == nil {
		return
	}
	callbackStore(unsafe.Pointer(tok), w)
	C.xdg_activation_token_v1_add_listener(tok, &C.gio_xdg_activation_token_v1_listener, unsafe.Pointer(tok))
	if s := w.disp.seat; s != nil && s.serial != 0 {
		C.xdg_activation_token_v1_set_serial(tok, s.serial, s.seat)
	}
	C.xdg_activation_token_v1_set_surface(tok, w.surf)
	C.xdg_activation_token_v1_commit(tok)
	C.wl_display_flush(w.disp.disp)
}

//export gio_onActivationTokenDone
func gio_onActivationTokenDone(data unsafe.Pointer, tok *C.struct_xdg_activation_token_v1, token *C.char) {
	w, _ := callbackLoad(data).(*window)
	callbackDelete(data)
	defer C.xdg_activation_token_v1_destroy(tok)
	if w == nil || w.surf == nil || w.disp.lt.activation == nil {
		return
	}
	C.xdg_activation_v1_activate(w.disp.lt.activation, token, w.surf)
}
