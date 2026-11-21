// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd) && !nowayland

// LeanTerm patch: listeners for xdg-activation (see os_wayland_leanterm.go).

#include <wayland-client.h>
#include "wayland_xdg_activation.h"
#include "_cgo_export.h"

const struct xdg_activation_token_v1_listener gio_xdg_activation_token_v1_listener = {
	.done = (void (*)(void *, struct xdg_activation_token_v1 *, const char *))gio_onActivationTokenDone,
};

static void gio_lt_drag_target(void *data, struct wl_data_source *src, const char *mime) {}
static void gio_lt_drag_noop(void *data, struct wl_data_source *src) {}
static void gio_lt_drag_action(void *data, struct wl_data_source *src, uint32_t action) {}

const struct wl_data_source_listener gio_lt_drag_source_listener = {
	.target = gio_lt_drag_target,
	.send = (void (*)(void *, struct wl_data_source *, const char *, int32_t))gio_onLtDragSend,
	.cancelled = gio_onLtDragEnd,
	.dnd_drop_performed = gio_lt_drag_noop,
	.dnd_finished = gio_onLtDragEnd,
	.action = gio_lt_drag_action,
};
