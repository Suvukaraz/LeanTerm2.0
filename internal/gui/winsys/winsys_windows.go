//go:build windows

package winsys

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"gioui.org/app"
	"golang.org/x/sys/windows"
)

var (
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")
	ole32    = windows.NewLazySystemDLL("ole32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procSetCurrentProcessExplicitAppUserModelID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	procSHGetPropertyStoreForWindow             = shell32.NewProc("SHGetPropertyStoreForWindow")
	procAllowSetForegroundWindow                = user32.NewProc("AllowSetForegroundWindow")
	procSendMessageW                            = user32.NewProc("SendMessageW")
	procLoadImageW                              = user32.NewProc("LoadImageW")
	procGetSystemMetrics                        = user32.NewProc("GetSystemMetrics")
	procCoInitializeEx                          = ole32.NewProc("CoInitializeEx")
	procCoUninitialize                          = ole32.NewProc("CoUninitialize")
	procCoCreateInstance                        = ole32.NewProc("CoCreateInstance")
	procGetModuleHandleW                        = kernel32.NewProc("GetModuleHandleW")
)

const (
	asfwAny            = 0xFFFFFFFF
	wmSetIcon          = 0x0080
	iconSmall          = 0
	iconBig            = 1
	imageIcon          = 1
	lrShared           = 0x8000
	smCxIcon           = 11
	smCyIcon           = 12
	smCxSmIcon         = 49
	smCySmIcon         = 50
	coinitApartment    = 0x2
	clsctxInprocServer = 0x1
	stgmReadWrite      = 0x2
	vtLPWStr           = 31
	sFalse             = 1
	rpcEChangedMode    = 0x80010106
)

var (
	// PKEY_AppUserModel_* (propkey.h): {9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3}.
	pkeyAppUserModel = windows.GUID{Data1: 0x9F4C2855, Data2: 0x9F79, Data3: 0x4B39, Data4: [8]byte{0xA8, 0xD0, 0xE1, 0xD4, 0x2D, 0xE1, 0xD5, 0xF3}}

	iidIPropertyStore = windows.GUID{Data1: 0x886D8EEB, Data2: 0x8CF2, Data3: 0x4446, Data4: [8]byte{0x8D, 0x02, 0xCD, 0xBA, 0x1D, 0xBD, 0xCF, 0x99}}
	iidIPersistFile   = windows.GUID{Data1: 0x0000010B, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIShellLinkW    = windows.GUID{Data1: 0x000214F9, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	clsidShellLink    = windows.GUID{Data1: 0x00021401, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

// The PKEY_AppUserModel_* property ids.
const (
	pidRelaunchCommand             = 2
	pidRelaunchIconResource        = 3
	pidRelaunchDisplayNameResource = 4
	pidID                          = 5
)

type propertyKey struct {
	fmtid windows.GUID
	pid   uint32
}

// propVariant is a PROPVARIANT holding a VT_LPWSTR.
type propVariant struct {
	vt         uint16
	r1, r2, r3 uint16
	val        uintptr
	pad        uintptr
}

// comObject is a COM interface pointer: its first word is the vtable.
type comObject struct {
	vtbl *[16]uintptr
}

func (o *comObject) call(method int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[method], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

// The IUnknown and IPropertyStore methods by vtable index.
const (
	mQueryInterface = 0
	mRelease        = 2
	mPSSetValue     = 6
	mPSCommit       = 7
	mPFLoad         = 5
	mPFSave         = 6
)

func (o *comObject) release() { o.call(mRelease) }

func (o *comObject) queryInterface(iid *windows.GUID) (*comObject, error) {
	var out *comObject
	if hr := o.call(mQueryInterface, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out))); hr != 0 || out == nil {
		return nil, fmt.Errorf("QueryInterface: HRESULT %#x", uint32(hr))
	}
	return out, nil
}

// setString sets a string property of a property store.
func setString(ps *comObject, pid uint32, value string) error {
	s, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	key := propertyKey{fmtid: pkeyAppUserModel, pid: pid}
	pv := propVariant{vt: vtLPWStr, val: uintptr(unsafe.Pointer(s))}
	hr := ps.call(mPSSetValue, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&pv)))
	runtime.KeepAlive(s)
	if hr != 0 {
		return fmt.Errorf("IPropertyStore.SetValue: HRESULT %#x", uint32(hr))
	}
	return nil
}

// withCOM runs fn on a locked thread with COM initialized.
func withCOM(fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartment)
	if hr == 0 || hr == sFalse {
		defer procCoUninitialize.Call()
	} else if uint32(hr) != rpcEChangedMode {
		return fmt.Errorf("CoInitializeEx: HRESULT %#x", uint32(hr))
	}
	return fn()
}

func setProcessAppID(a *App) {
	id, err := windows.UTF16PtrFromString(a.ID)
	if err != nil {
		return
	}
	_, _, _ = procSetCurrentProcessExplicitAppUserModelID.Call(uintptr(unsafe.Pointer(id)))
}

func allowActivation() {
	_, _, _ = procAllowSetForegroundWindow.Call(asfwAny)
}

// relaunch is how the taskbar starts a pinned PureEdit: its own executable
// next to this one, else this one with -pureedit.
func relaunch(a *App) (cmd, icon string) {
	exe, err := os.Executable()
	if err != nil {
		return "", ""
	}
	own := filepath.Join(filepath.Dir(exe), a.Exe)
	if _, err := os.Stat(own); err == nil {
		return `"` + own + `"`, own + ",0"
	}
	if strings.EqualFold(filepath.Base(exe), PureEdit.Exe) || a != PureEdit {
		return `"` + exe + `"`, exe + ",0"
	}
	return `"` + exe + `" -pureedit`, fmt.Sprintf("%s,-%d", exe, a.IconResource)
}

func view(a *App, e app.ViewEvent) {
	ev, ok := e.(app.Win32ViewEvent)
	if !ok || !ev.Valid() {
		return
	}
	hwnd := ev.HWND
	go func() {
		setWindowIcon(hwnd, a)
		_ = withCOM(func() error { return setWindowAppID(hwnd, a) })
	}()
}

// setWindowAppID gives the window its application's AppUserModelID and how
// a pinned window is started again.
func setWindowAppID(hwnd uintptr, a *App) error {
	var ps *comObject
	hr, _, _ := procSHGetPropertyStoreForWindow.Call(hwnd, uintptr(unsafe.Pointer(&iidIPropertyStore)), uintptr(unsafe.Pointer(&ps)))
	if hr != 0 || ps == nil {
		return fmt.Errorf("SHGetPropertyStoreForWindow: HRESULT %#x", uint32(hr))
	}
	defer ps.release()
	cmd, icon := relaunch(a)
	if err := setString(ps, pidID, a.ID); err != nil {
		return err
	}
	if cmd != "" {
		_ = setString(ps, pidRelaunchCommand, cmd)
		_ = setString(ps, pidRelaunchIconResource, icon)
		_ = setString(ps, pidRelaunchDisplayNameResource, a.Name)
	}
	ps.call(mPSCommit)
	return nil
}

// setWindowIcon sets the window's icons from the executable's resources.
func setWindowIcon(hwnd uintptr, a *App) {
	module, _, _ := procGetModuleHandleW.Call(0)
	if module == 0 {
		return
	}
	load := func(cx, cy int) uintptr {
		w, _, _ := procGetSystemMetrics.Call(uintptr(cx))
		h, _, _ := procGetSystemMetrics.Call(uintptr(cy))
		icon, _, _ := procLoadImageW.Call(module, uintptr(a.IconResource), imageIcon, w, h, lrShared)
		return icon
	}
	if big := load(smCxIcon, smCyIcon); big != 0 {
		_, _, _ = procSendMessageW.Call(hwnd, wmSetIcon, iconBig, big)
	}
	if small := load(smCxSmIcon, smCySmIcon); small != 0 {
		_, _, _ = procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, small)
	}
}

// SetShortcutAppID writes the AppUserModelID into a shortcut (.lnk), so that
// the taskbar puts the windows started from it, and the shortcut pinned
// there, into one group. The installer calls it (pureterm.exe
// -set-shortcut-appid).
func SetShortcutAppID(lnk, id string) error {
	path, err := windows.UTF16PtrFromString(lnk)
	if err != nil {
		return err
	}
	return withCOM(func() error {
		var link *comObject
		hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, clsctxInprocServer,
			uintptr(unsafe.Pointer(&iidIShellLinkW)), uintptr(unsafe.Pointer(&link)))
		if hr != 0 || link == nil {
			return fmt.Errorf("CoCreateInstance(ShellLink): HRESULT %#x", uint32(hr))
		}
		defer link.release()
		pf, err := link.queryInterface(&iidIPersistFile)
		if err != nil {
			return err
		}
		defer pf.release()
		if hr := pf.call(mPFLoad, uintptr(unsafe.Pointer(path)), stgmReadWrite); hr != 0 {
			return fmt.Errorf("load %s: HRESULT %#x", lnk, uint32(hr))
		}
		ps, err := link.queryInterface(&iidIPropertyStore)
		if err != nil {
			return err
		}
		defer ps.release()
		if err := setString(ps, pidID, id); err != nil {
			return err
		}
		if hr := ps.call(mPSCommit); hr != 0 {
			return fmt.Errorf("IPropertyStore.Commit: HRESULT %#x", uint32(hr))
		}
		if hr := pf.call(mPFSave, 0, 1); hr != 0 {
			return fmt.Errorf("save %s: HRESULT %#x", lnk, uint32(hr))
		}
		runtime.KeepAlive(path)
		return nil
	})
}
