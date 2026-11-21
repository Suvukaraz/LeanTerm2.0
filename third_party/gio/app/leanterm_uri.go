// SPDX-License-Identifier: Unlicense OR MIT

// LeanTerm patch: the file list of a drop (text/uri-list).

package app

import (
	"net/url"
	"os"
	"strings"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
)

// dragOutside is where a drag that left the window moves the pointer.
var dragOutside = f32.Point{X: -1 << 20, Y: -1 << 20}

// dragMove reports the position of a drag over the window as a pointer
// move: the program sees what is under the dragged files (hover feedback,
// and the target of the drop that follows).
func dragMove(process func(e event.Event), pos f32.Point) {
	process(pointer.Event{Kind: pointer.Move, Position: pos, Source: pointer.Mouse})
}

const uriListMime = "text/uri-list"

// parseURIList returns the local paths of a text/uri-list (RFC 2483):
// file URIs of this host, one per line, comments starting with '#'.
func parseURIList(s string) []string {
	var paths []string
	host, _ := os.Hostname()
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme != "file" {
			continue
		}
		if u.Host != "" && u.Host != "localhost" && !strings.EqualFold(u.Host, host) {
			continue
		}
		if u.Path != "" {
			paths = append(paths, u.Path)
		}
	}
	return paths
}
