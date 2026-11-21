package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// dragOut continues a drag of file browser rows as a drag of files to the
// desktop or a file manager once the pointer leaves the window. Remote
// files are downloaded into a temporary folder first (when the target asks
// for them, or right away on Windows).
func (u *ui) dragOut() {
	t := u.activeTab()
	if t == nil || t.files == nil || !t.sftpOpen {
		u.dragOutStarted = false
		return
	}
	paths, sftpID, ok := t.files.DragOut()
	if !ok {
		u.dragOutStarted = false
		return
	}
	p := u.lastPtr
	inside := p.X >= 0 && p.Y >= 0 && p.X < u.winWidth && p.Y < u.winHeight
	if u.dragOutStarted || inside {
		return
	}
	u.dragOutStarted = true
	paths = append([]string(nil), paths...)
	var once sync.Once
	var local []string
	files := func() []string {
		once.Do(func() {
			if sftpID == "" {
				local = paths
				return
			}
			local = u.downloadForDrag(sftpID, paths)
		})
		return local
	}
	if sftpID != "" {
		go files() // start downloading while the user is still dragging
	}
	u.w.StartFileDrag(files)
}

// downloadForDrag downloads remote files into a new temporary folder and
// returns their local paths (nil on failure, which is reported).
func (u *ui) downloadForDrag(sftpID string, remote []string) []string {
	base := filepath.Join(os.TempDir(), "leanterm-drag")
	cleanDragFolders(base)
	if err := os.MkdirAll(base, 0o700); err != nil {
		u.postNotice("Drag to the desktop: " + err.Error())
		return nil
	}
	dir, err := os.MkdirTemp(base, "files-")
	if err != nil {
		u.postNotice("Drag to the desktop: " + err.Error())
		return nil
	}
	if _, err := u.sftpSvc.Download(sftpID, remote, dir); err != nil {
		u.postNotice("Drag to the desktop: the download failed: " + err.Error())
		return nil
	}
	var out []string
	for _, r := range remote {
		name := r[strings.LastIndex(r, "/")+1:]
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

// cleanDragFolders removes the temporary folders of drags older than a day
// (the target has copied the files long ago).
func cleanDragFolders(base string) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, e := range entries {
		if fi, err := e.Info(); err == nil && time.Since(fi.ModTime()) > 24*time.Hour {
			_ = os.RemoveAll(filepath.Join(base, e.Name()))
		}
	}
}
