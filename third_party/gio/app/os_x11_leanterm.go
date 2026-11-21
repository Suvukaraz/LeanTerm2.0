// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd || openbsd) && !nox11

// LeanTerm patch: files dropped on a window (XDND, delivered as
// FileDropEvent), the window's icon (_NET_WM_ICON), its process
// (_NET_WM_PID) and activation with a startup notification id.

package app

/*
#cgo freebsd openbsd CFLAGS: -I/usr/X11R6/include -I/usr/local/include
#cgo freebsd openbsd LDFLAGS: -L/usr/X11R6/lib -L/usr/local/lib -lX11
#cgo linux pkg-config: x11

#include <stdlib.h>
#include <string.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

static long gio_xdnd_get(XClientMessageEvent *e, int i) { return e->data.l[i]; }

static void gio_xdnd_send(Display *dpy, Window to, Atom type, long l0, long l1, long l2, long l3, long l4) {
	XEvent xev = {0};
	xev.xclient.type = ClientMessage;
	xev.xclient.display = dpy;
	xev.xclient.window = to;
	xev.xclient.message_type = type;
	xev.xclient.format = 32;
	xev.xclient.data.l[0] = l0;
	xev.xclient.data.l[1] = l1;
	xev.xclient.data.l[2] = l2;
	xev.xclient.data.l[3] = l3;
	xev.xclient.data.l[4] = l4;
	XSendEvent(dpy, to, False, NoEventMask, &xev);
	XFlush(dpy);
}

// gio_net_message sends a root window message (EWMH: _NET_ACTIVE_WINDOW).
static void gio_net_message(Display *dpy, Window w, Atom type, long l0, long l1, long l2) {
	XEvent xev = {0};
	xev.xclient.type = ClientMessage;
	xev.xclient.display = dpy;
	xev.xclient.window = w;
	xev.xclient.message_type = type;
	xev.xclient.format = 32;
	xev.xclient.data.l[0] = l0;
	xev.xclient.data.l[1] = l1;
	xev.xclient.data.l[2] = l2;
	XSendEvent(dpy, DefaultRootWindow(dpy), False, SubstructureNotifyMask | SubstructureRedirectMask, &xev);
	XFlush(dpy);
}

// gio_startup_message sends a startup notification message (in 20 byte
// pieces, the first with _NET_STARTUP_INFO_BEGIN).
static void gio_startup_message(Display *dpy, Window w, Atom begin, Atom more, const char *msg) {
	size_t n = strlen(msg) + 1;
	for (size_t off = 0; off < n; off += 20) {
		XEvent xev = {0};
		xev.xclient.type = ClientMessage;
		xev.xclient.display = dpy;
		xev.xclient.window = w;
		xev.xclient.message_type = off == 0 ? begin : more;
		xev.xclient.format = 8;
		size_t k = n - off < 20 ? n - off : 20;
		memcpy(xev.xclient.data.b, msg + off, k);
		XSendEvent(dpy, DefaultRootWindow(dpy), False, PropertyChangeMask, &xev);
	}
	XFlush(dpy);
}
*/
import "C"

import (
	"image"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/image/draw"

	"gioui.org/f32"
)

// x11LeanTerm is the window's state of the LeanTerm patch.
type x11LeanTerm struct {
	atoms struct {
		aware, enter, position, status, leave, drop, finished C.Atom
		selection, actionCopy, typeList, uriList, data        C.Atom
		wmIcon, wmPid, startupID, startupBegin, startupMore   C.Atom
	}
	// The drag over the window.
	source   C.Window
	version  int
	accepted bool
	dropping bool
	pos      f32.Point
}

// initLeanTerm sets the window's XDND, icon and process properties.
func (w *x11Window) initLeanTerm(cnf Config) {
	a := &w.lt.atoms
	a.aware = w.atom("XdndAware", false)
	a.enter = w.atom("XdndEnter", false)
	a.position = w.atom("XdndPosition", false)
	a.status = w.atom("XdndStatus", false)
	a.leave = w.atom("XdndLeave", false)
	a.drop = w.atom("XdndDrop", false)
	a.finished = w.atom("XdndFinished", false)
	a.selection = w.atom("XdndSelection", false)
	a.actionCopy = w.atom("XdndActionCopy", false)
	a.typeList = w.atom("XdndTypeList", false)
	a.uriList = w.atom(uriListMime, false)
	a.data = w.atom("GIO_XDND_DATA", false)
	a.wmIcon = w.atom("_NET_WM_ICON", false)
	a.wmPid = w.atom("_NET_WM_PID", false)
	a.startupID = w.atom("_NET_STARTUP_ID", false)
	a.startupBegin = w.atom("_NET_STARTUP_INFO_BEGIN", false)
	a.startupMore = w.atom("_NET_STARTUP_INFO", false)

	version := C.long(5)
	C.XChangeProperty(w.x, w.xw, a.aware, C.XA_ATOM, 32, C.PropModeReplace, (*C.uchar)(unsafe.Pointer(&version)), 1)
	pid := C.long(os.Getpid())
	C.XChangeProperty(w.x, w.xw, a.wmPid, C.XA_CARDINAL, 32, C.PropModeReplace, (*C.uchar)(unsafe.Pointer(&pid)), 1)
	if cnf.icon != nil {
		w.setIcon(cnf.icon)
	}
}

// setIcon sets _NET_WM_ICON: the image in the sizes desktops use.
func (w *x11Window) setIcon(img image.Image) {
	var data []C.long
	for _, size := range []int{16, 24, 32, 48, 64, 128, 256} {
		dst := image.NewNRGBA(image.Rect(0, 0, size, size))
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
		data = append(data, C.long(size), C.long(size))
		for i := 0; i < len(dst.Pix); i += 4 {
			p := dst.Pix[i : i+4]
			data = append(data, C.long(uint32(p[3])<<24|uint32(p[0])<<16|uint32(p[1])<<8|uint32(p[2])))
		}
	}
	C.XChangeProperty(w.x, w.xw, w.lt.atoms.wmIcon, C.XA_CARDINAL, 32, C.PropModeReplace,
		(*C.uchar)(unsafe.Pointer(&data[0])), C.int(len(data)))
}

// activate raises the window. token is a startup notification id
// (DESKTOP_STARTUP_ID) of a process that handed its work over: the window
// takes it over, which ends the launch feedback and lets the window manager
// allow the activation.
func (w *x11Window) activate(token string) {
	a := &w.lt.atoms
	var ts C.long
	if token != "" {
		t := C.CString(token)
		C.XChangeProperty(w.x, w.xw, a.startupID, w.atoms.utf8string, 8, C.PropModeReplace, (*C.uchar)(unsafe.Pointer(t)), C.int(len(token)))
		msg := C.CString(`remove: ID="` + strings.ReplaceAll(strings.ReplaceAll(token, `\`, `\\`), `"`, `\"`) + `"`)
		C.gio_startup_message(w.x, w.xw, a.startupBegin, a.startupMore, msg)
		C.free(unsafe.Pointer(msg))
		C.free(unsafe.Pointer(t))
		if i := strings.LastIndex(token, "_TIME"); i >= 0 {
			if v, err := strconv.ParseUint(token[i+5:], 10, 32); err == nil {
				ts = C.long(v)
			}
		}
	}
	C.XMapRaised(w.x, w.xw)
	// Source 2 (a pager or taskbar, on behalf of the user): window
	// managers do not hold such requests back as focus stealing.
	C.gio_net_message(w.x, w.xw, w.atoms.wmActiveWindow, 2, ts, 0)
}

// xdndMessage handles the XDND client messages; it reports whether ev was one.
func (w *x11Window) xdndMessage(ev *C.XClientMessageEvent) bool {
	a := &w.lt.atoms
	l := func(i int) C.long { return C.gio_xdnd_get(ev, C.int(i)) }
	switch ev.message_type {
	case a.enter:
		w.lt.source = C.Window(l(0))
		w.lt.version = int(uint64(l(1)) >> 24)
		w.lt.accepted, w.lt.dropping = false, false
		var types []C.Atom
		if l(1)&1 != 0 {
			types = w.xdndTypeList(w.lt.source)
		} else {
			for i := 2; i <= 4; i++ {
				types = append(types, C.Atom(l(i)))
			}
		}
		for _, t := range types {
			if t == a.uriList {
				w.lt.accepted = true
			}
		}
	case a.position:
		if C.Window(l(0)) != w.lt.source {
			return true
		}
		x, y := C.int(uint64(l(2))>>16&0xffff), C.int(uint64(l(2))&0xffff)
		var wx, wy C.int
		var child C.Window
		C.XTranslateCoordinates(w.x, C.XDefaultRootWindow(w.x), w.xw, x, y, &wx, &wy, &child)
		w.lt.pos = f32.Point{X: float32(wx), Y: float32(wy)}
		dragMove(w.ProcessEvent, w.lt.pos)
		accept, action := C.long(0), C.long(C.None)
		if w.lt.accepted {
			accept, action = 1, C.long(a.actionCopy)
		}
		C.gio_xdnd_send(w.x, w.lt.source, a.status, C.long(w.xw), accept, 0, 0, action)
	case a.leave:
		if C.Window(l(0)) == w.lt.source && !w.lt.dropping {
			w.lt.source, w.lt.accepted = 0, false
			dragMove(w.ProcessEvent, dragOutside)
		}
	case a.drop:
		if C.Window(l(0)) != w.lt.source {
			return true
		}
		if !w.lt.accepted {
			w.xdndFinish(false)
			return true
		}
		w.lt.dropping = true
		C.XConvertSelection(w.x, a.selection, a.uriList, a.data, w.xw, C.Time(l(2)))
		C.XFlush(w.x)
	default:
		return false
	}
	return true
}

// xdndTypeList reads the source's XdndTypeList (more than three types).
func (w *x11Window) xdndTypeList(src C.Window) []C.Atom {
	var actual C.Atom
	var format C.int
	var n, after C.ulong
	var data *C.uchar
	if C.XGetWindowProperty(w.x, src, w.lt.atoms.typeList, 0, 1024, C.False, C.XA_ATOM, &actual, &format, &n, &after, &data) != C.Success || data == nil {
		return nil
	}
	defer C.XFree(unsafe.Pointer(data))
	if format != 32 {
		return nil
	}
	longs := unsafe.Slice((*C.long)(unsafe.Pointer(data)), int(n))
	out := make([]C.Atom, len(longs))
	for i, v := range longs {
		out[i] = C.Atom(v)
	}
	return out
}

// xdndSelection takes the dropped file list (SelectionNotify of
// XdndSelection); it reports whether ev was one.
func (w *x11Window) xdndSelection(ev *C.XSelectionEvent) bool {
	a := &w.lt.atoms
	if ev.selection != a.selection {
		return false
	}
	if !w.lt.dropping {
		return true
	}
	var paths []string
	if ev.property != C.None {
		var actual C.Atom
		var format C.int
		var n, after C.ulong
		var data *C.uchar
		if C.XGetWindowProperty(w.x, w.xw, ev.property, 0, 4<<20, C.True, C.AnyPropertyType, &actual, &format, &n, &after, &data) == C.Success && data != nil {
			if format == 8 {
				paths = parseURIList(C.GoStringN((*C.char)(unsafe.Pointer(data)), C.int(n)))
			}
			C.XFree(unsafe.Pointer(data))
		}
	}
	pos := w.lt.pos
	w.xdndFinish(len(paths) > 0)
	if len(paths) > 0 {
		w.ProcessEvent(FileDropEvent{Paths: paths, Position: pos})
	}
	return true
}

// xdndFinish tells the source that the drop is done.
func (w *x11Window) xdndFinish(ok bool) {
	a := &w.lt.atoms
	if w.lt.source != 0 {
		accepted, action := C.long(0), C.long(C.None)
		if ok {
			accepted, action = 1, C.long(a.actionCopy)
		}
		C.gio_xdnd_send(w.x, w.lt.source, a.finished, C.long(w.xw), accepted, action, 0, 0)
	}
	w.lt.source, w.lt.accepted, w.lt.dropping = 0, false, false
}
