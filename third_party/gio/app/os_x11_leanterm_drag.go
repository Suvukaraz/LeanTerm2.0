// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd || openbsd) && !nox11

// LeanTerm patch: dragging files out of a window to other applications
// (XDND source, see StartFileDrag).

package app

/*
#cgo freebsd openbsd CFLAGS: -I/usr/X11R6/include -I/usr/local/include
#cgo freebsd openbsd LDFLAGS: -L/usr/X11R6/lib -L/usr/local/lib -lX11
#cgo linux pkg-config: x11

#include <stdlib.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>

static void gio_xdnd_src_send(Display *dpy, Window to, Atom type, long l0, long l1, long l2, long l3, long l4) {
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

static void gio_xdnd_src_notify(Display *dpy, XSelectionRequestEvent *r, Atom property) {
	XEvent xev = {0};
	xev.xselection.type = SelectionNotify;
	xev.xselection.display = r->display;
	xev.xselection.requestor = r->requestor;
	xev.xselection.selection = r->selection;
	xev.xselection.target = r->target;
	xev.xselection.property = property;
	xev.xselection.time = r->time;
	XSendEvent(dpy, r->requestor, False, NoEventMask, &xev);
	XFlush(dpy);
}
*/
import "C"

import (
	"sync"
	"time"
	"unsafe"
)

// x11DragSource is a drag of files from the window to other applications.
type x11DragSource struct {
	files    *fileList
	target   C.Window
	accepted bool
	dropped  bool
}

// fileList fetches the dragged files once (it may download them) and
// formats them as text/uri-list.
type fileList struct {
	once  sync.Once
	files func() []string
	list  string
}

func (f *fileList) uriList() string {
	f.once.Do(func() { f.list = formatURIList(f.files()) })
	return f.list
}

// startFileDrag starts dragging files while the primary button is held.
func (w *x11Window) startFileDrag(files func() []string) {
	fl := &fileList{files: files}
	start := func() {
		w.lt.src = &x11DragSource{files: fl}
		C.XSetSelectionOwner(w.x, w.lt.atoms.selection, w.xw, C.CurrentTime)
		C.XFlush(w.x)
	}
	select {
	case w.dispatchFns <- start:
		w.invalidate()
	default:
	}
}

// xdndAware returns the XDND version a window accepts (0: none).
func (w *x11Window) xdndAware(win C.Window) int {
	var actual C.Atom
	var format C.int
	var n, after C.ulong
	var data *C.uchar
	if C.XGetWindowProperty(w.x, win, w.lt.atoms.aware, 0, 1, C.False, C.XA_ATOM, &actual, &format, &n, &after, &data) != C.Success || data == nil {
		return 0
	}
	defer C.XFree(unsafe.Pointer(data))
	if n == 0 || format != 32 {
		return 0
	}
	return int(*(*C.long)(unsafe.Pointer(data)))
}

// xdndTargetAt finds the XDND aware window under the root position.
func (w *x11Window) xdndTargetAt(x, y C.int) (C.Window, int) {
	root := C.XDefaultRootWindow(w.x)
	cur := root
	for depth := 0; depth < 32; depth++ {
		if cur != root {
			if v := w.xdndAware(cur); v > 0 {
				return cur, v
			}
		}
		var cx, cy C.int
		var child C.Window
		if C.XTranslateCoordinates(w.x, root, cur, x, y, &cx, &cy, &child) == 0 || child == 0 {
			return 0, 0
		}
		cur = child
	}
	return 0, 0
}

// dragMotion follows the pointer of a drag from the window.
func (w *x11Window) dragMotion(m *C.XMotionEvent) {
	src := w.lt.src
	if src == nil || src.dropped {
		return
	}
	a := &w.lt.atoms
	target, version := w.xdndTargetAt(m.x_root, m.y_root)
	if target == w.xw {
		target = 0 // not onto itself
	}
	if target != src.target {
		if src.target != 0 {
			C.gio_xdnd_src_send(w.x, src.target, a.leave, C.long(w.xw), 0, 0, 0, 0)
		}
		src.target, src.accepted = target, false
		if target != 0 {
			C.gio_xdnd_src_send(w.x, target, a.enter, C.long(w.xw), C.long(min(version, 5))<<24, C.long(a.uriList), 0, 0)
		}
	}
	if target != 0 {
		C.gio_xdnd_src_send(w.x, target, a.position, C.long(w.xw), 0, C.long(m.x_root)<<16|C.long(m.y_root), C.long(m.time), C.long(a.actionCopy))
	}
}

// dragRelease ends a drag: the files are dropped where the target took them.
func (w *x11Window) dragRelease(b *C.XButtonEvent) {
	src := w.lt.src
	if src == nil || src.dropped {
		return
	}
	a := &w.lt.atoms
	if src.target != 0 && src.accepted {
		src.dropped = true
		C.gio_xdnd_src_send(w.x, src.target, a.drop, C.long(w.xw), 0, C.long(b.time), 0, 0)
		// A target that never finishes does not keep the drag forever.
		time.AfterFunc(time.Minute, func() {
			select {
			case w.dispatchFns <- func() {
				if w.lt.src == src {
					w.endFileDrag()
				}
			}:
				w.invalidate()
			default:
			}
		})
		return
	}
	if src.target != 0 {
		C.gio_xdnd_src_send(w.x, src.target, a.leave, C.long(w.xw), 0, 0, 0, 0)
	}
	w.endFileDrag()
}

func (w *x11Window) endFileDrag() {
	w.lt.src = nil
	C.XSetSelectionOwner(w.x, w.lt.atoms.selection, C.None, C.CurrentTime)
	C.XFlush(w.x)
}

// dragMessage handles XdndStatus and XdndFinished of a drag from the window.
func (w *x11Window) dragMessage(ev *C.XClientMessageEvent) bool {
	a := &w.lt.atoms
	src := w.lt.src
	l := (*[5]C.long)(unsafe.Pointer(&ev.data))
	switch ev.message_type {
	case a.status:
		if src != nil && C.Window(l[0]) == src.target {
			src.accepted = l[1]&1 != 0
		}
	case a.finished:
		if src != nil && src.dropped {
			w.endFileDrag()
		}
	default:
		return false
	}
	return true
}

// dragSelectionRequest answers a target's request for the dragged files.
// Fetching them may take time (downloads): it runs in the background.
func (w *x11Window) dragSelectionRequest(r *C.XSelectionRequestEvent) bool {
	a := &w.lt.atoms
	if r.selection != a.selection {
		return false
	}
	src := w.lt.src
	prop := r.property
	if prop == C.None {
		prop = r.target
	}
	switch {
	case src == nil:
		C.gio_xdnd_src_notify(w.x, r, C.None)
	case r.target == w.atoms.targets:
		targets := [...]C.long{C.long(w.atoms.targets), C.long(a.uriList)}
		C.XChangeProperty(w.x, r.requestor, prop, w.atoms.atom, 32, C.PropModeReplace, (*C.uchar)(unsafe.Pointer(&targets[0])), C.int(len(targets)))
		C.gio_xdnd_src_notify(w.x, r, prop)
	case r.target == a.uriList:
		req := *r
		go func() {
			list := []byte(src.files.uriList())
			if len(list) == 0 {
				C.gio_xdnd_src_notify(w.x, &req, C.None)
				return
			}
			C.XChangeProperty(w.x, req.requestor, prop, a.uriList, 8, C.PropModeReplace, (*C.uchar)(unsafe.Pointer(&list[0])), C.int(len(list)))
			C.gio_xdnd_src_notify(w.x, &req, prop)
		}()
	default:
		C.gio_xdnd_src_notify(w.x, r, C.None)
	}
	return true
}
