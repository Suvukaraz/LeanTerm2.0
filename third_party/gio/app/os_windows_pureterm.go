// SPDX-License-Identifier: Unlicense OR MIT

//go:build windows

// PureTerm patch: dragging files out of a window to other applications
// (the shell's data object and default drop source, see StartFileDrag).

package app

import (
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	syswin "golang.org/x/sys/windows"
)

var (
	ltOle32             = syswin.NewLazySystemDLL("ole32.dll")
	ltShell32           = syswin.NewLazySystemDLL("shell32.dll")
	ltUser32            = syswin.NewLazySystemDLL("user32.dll")
	procOleInitialize   = ltOle32.NewProc("OleInitialize")
	procOleUninitialize = ltOle32.NewProc("OleUninitialize")
	procSHParseName     = ltShell32.NewProc("SHParseDisplayName")
	procSHCreateDataObj = ltShell32.NewProc("SHCreateDataObject")
	procSHDoDragDrop    = ltShell32.NewProc("SHDoDragDrop")
	procILFree          = ltShell32.NewProc("ILFree")
	procILClone         = ltShell32.NewProc("ILClone")
	procILRemoveLastID  = ltShell32.NewProc("ILRemoveLastID")
	procILFindLastID    = ltShell32.NewProc("ILFindLastID")
	procGetCursorPos    = ltUser32.NewProc("GetCursorPos")
	procScreenToClient  = ltUser32.NewProc("ScreenToClient")
	procPostMessage     = ltUser32.NewProc("PostMessageW")
	iidIDataObject      = syswin.GUID{Data1: 0x0000010e, Data2: 0, Data3: 0, Data4: [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	ltDropEffectCopy    = uintptr(1)
	ltWMLButtonUp       = uintptr(0x0202)
)

// startFileDrag starts dragging files while the primary button is held. The
// drag runs on a thread of its own (OLE's modal drag loop); the files are
// fetched there first, as the shell needs them to exist.
func (w *window) startFileDrag(files func() []string) {
	hwnd := uintptr(w.hwnd)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if hr, _, _ := procOleInitialize.Call(0); int32(hr) < 0 {
			return
		}
		defer procOleUninitialize.Call()
		paths := files()
		if len(paths) > 0 {
			shellDrag(hwnd, paths)
		}
		// The drag took the mouse capture: the window gets the release it
		// missed, so it does not keep the button pressed.
		var pt struct{ X, Y int32 }
		procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
		procPostMessage.Call(hwnd, ltWMLButtonUp, 0, uintptr(uint16(pt.X))|uintptr(uint16(pt.Y))<<16)
	}()
}

// shellDrag drags paths of one folder (the others are left out) with the
// shell's data object, as Explorer does.
func shellDrag(hwnd uintptr, paths []string) {
	dir := strings.ToLower(filepath.Dir(paths[0]))
	var full []uintptr
	defer func() {
		for _, p := range full {
			procILFree.Call(p)
		}
	}()
	for _, p := range paths {
		if strings.ToLower(filepath.Dir(p)) != dir {
			continue
		}
		name, err := syswin.UTF16PtrFromString(p)
		if err != nil {
			continue
		}
		var pidl uintptr
		if hr, _, _ := procSHParseName.Call(uintptr(unsafe.Pointer(name)), 0, uintptr(unsafe.Pointer(&pidl)), 0, 0); int32(hr) < 0 || pidl == 0 {
			continue
		}
		full = append(full, pidl)
	}
	if len(full) == 0 {
		return
	}
	folder, _, _ := procILClone.Call(full[0])
	if folder == 0 {
		return
	}
	defer procILFree.Call(folder)
	procILRemoveLastID.Call(folder)
	children := make([]uintptr, len(full))
	for i, p := range full {
		children[i], _, _ = procILFindLastID.Call(p)
	}
	var obj *struct{ vtbl *[3]uintptr }
	if hr, _, _ := procSHCreateDataObj.Call(folder, uintptr(len(children)), uintptr(unsafe.Pointer(&children[0])), 0,
		uintptr(unsafe.Pointer(&iidIDataObject)), uintptr(unsafe.Pointer(&obj))); int32(hr) < 0 || obj == nil {
		return
	}
	defer syscall.SyscallN(obj.vtbl[2], uintptr(unsafe.Pointer(obj))) // IUnknown.Release
	var effect uint32
	procSHDoDragDrop.Call(hwnd, uintptr(unsafe.Pointer(obj)), 0, ltDropEffectCopy, uintptr(unsafe.Pointer(&effect)))
	runtime.KeepAlive(children)
}
