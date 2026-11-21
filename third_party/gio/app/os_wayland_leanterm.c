// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd) && !nowayland

// LeanTerm patch: listeners for xdg-activation (see os_wayland_leanterm.go).

#include <wayland-client.h>
#include "wayland_xdg_activation.h"
#include "_cgo_export.h"

const struct xdg_activation_token_v1_listener gio_xdg_activation_token_v1_listener = {
	.done = (void (*)(void *, struct xdg_activation_token_v1 *, const char *))gio_onActivationTokenDone,
};
