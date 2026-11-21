// xdnd-drop drags a file list onto an X11 window with the XDND protocol, as a
// file manager does, for the CI's test of dropping files on LeanEdit and
// LeanTerm:
//
//   cc -o xdnd-drop build/ci/xdnd-drop.c -lX11
//   xdnd-drop <window id> <x> <y> 'file:///path/a.txt'
//
// x and y are root window coordinates. It exits 0 when the target finished
// the drop and accepted it.
#include <X11/Xlib.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static Display *dpy;
static Window src;

static void send(Window to, Atom type, long l0, long l1, long l2, long l3, long l4) {
	XEvent e = {0};
	e.xclient.type = ClientMessage;
	e.xclient.window = to;
	e.xclient.message_type = type;
	e.xclient.format = 32;
	e.xclient.data.l[0] = l0;
	e.xclient.data.l[1] = l1;
	e.xclient.data.l[2] = l2;
	e.xclient.data.l[3] = l3;
	e.xclient.data.l[4] = l4;
	XSendEvent(dpy, to, False, NoEventMask, &e);
	XFlush(dpy);
}

int main(int argc, char **argv) {
	if (argc < 5) {
		fprintf(stderr, "usage: xdnd-drop window x y uri-list\n");
		return 2;
	}
	Window target = strtoul(argv[1], 0, 0);
	int x = atoi(argv[2]), y = atoi(argv[3]);
	const char *list = argv[4];
	dpy = XOpenDisplay(0);
	if (!dpy) {
		fprintf(stderr, "cannot open the display\n");
		return 2;
	}
	src = XCreateSimpleWindow(dpy, DefaultRootWindow(dpy), 0, 0, 1, 1, 0, 0, 0);
	Atom enter = XInternAtom(dpy, "XdndEnter", 0), position = XInternAtom(dpy, "XdndPosition", 0),
	     drop = XInternAtom(dpy, "XdndDrop", 0), status = XInternAtom(dpy, "XdndStatus", 0),
	     finished = XInternAtom(dpy, "XdndFinished", 0), selection = XInternAtom(dpy, "XdndSelection", 0),
	     uriList = XInternAtom(dpy, "text/uri-list", 0), copy = XInternAtom(dpy, "XdndActionCopy", 0);
	XSetSelectionOwner(dpy, selection, src, CurrentTime);
	send(target, enter, src, 5L << 24, uriList, 0, 0);
	for (int i = 3; i >= 0; i--) {
		send(target, position, src, 0, ((long)(x - 10 * i) << 16) | (y - 10 * i), CurrentTime, copy);
		usleep(100000);
	}
	int accepted = -1, dropped = 0;
	for (int n = 0; n < 500; n++) {
		while (XPending(dpy)) {
			XEvent e;
			XNextEvent(dpy, &e);
			if (e.type == ClientMessage && e.xclient.message_type == status)
				accepted = e.xclient.data.l[1] & 1;
			if (e.type == ClientMessage && e.xclient.message_type == finished) {
				printf("drop finished, accepted=%ld\n", e.xclient.data.l[1] & 1);
				return (e.xclient.data.l[1] & 1) ? 0 : 1;
			}
			if (e.type == SelectionRequest) {
				XSelectionRequestEvent *r = &e.xselectionrequest;
				XEvent reply = {0};
				reply.xselection.type = SelectionNotify;
				reply.xselection.requestor = r->requestor;
				reply.xselection.selection = r->selection;
				reply.xselection.target = r->target;
				reply.xselection.time = r->time;
				reply.xselection.property = r->property;
				XChangeProperty(dpy, r->requestor, r->property, uriList, 8, PropModeReplace,
				                (const unsigned char *)list, strlen(list));
				XSendEvent(dpy, r->requestor, False, 0, &reply);
				XFlush(dpy);
			}
		}
		if (accepted == 1 && !dropped) {
			dropped = 1;
			send(target, drop, src, 0, CurrentTime, 0, 0);
		}
		usleep(20000);
	}
	fprintf(stderr, "no XdndFinished (status: %d)\n", accepted);
	return 1;
}
