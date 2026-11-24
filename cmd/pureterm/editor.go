package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"

	"github.com/Suvukaraz/PureTerm/internal/config"
	"github.com/Suvukaraz/PureTerm/internal/events"
	"github.com/Suvukaraz/PureTerm/internal/gui/dnd"
	"github.com/Suvukaraz/PureTerm/internal/gui/filedialog"
	"github.com/Suvukaraz/PureTerm/internal/gui/gtk"
	"github.com/Suvukaraz/PureTerm/internal/gui/pureedit"
	"github.com/Suvukaraz/PureTerm/internal/gui/prefs"
	"github.com/Suvukaraz/PureTerm/internal/gui/winsys"
	"github.com/Suvukaraz/PureTerm/internal/services"
	"github.com/Suvukaraz/PureTerm/internal/singleinstance"
)

// editorWindows is the native GUI's side of PureEdit: its window runs its
// own Gio event loop next to the main window's.
type editorWindows struct {
	store      *config.Store
	svc        *services.EditorService
	chooseFile func(title, dir string, folder bool) (string, error)
	// standalone: PureEdit is the only window; closing it quits.
	standalone bool
	onExit     func()

	mu      sync.Mutex
	w       *app.Window
	ed      *pureedit.App
	closing bool
	shapers map[string]*text.Shaper
	prefsW  *app.Window
	// startToken activates the first window of a standalone PureEdit (the
	// desktop's activation token of this process).
	startToken string
}

var _ services.EditorWindows = (*editorWindows)(nil)

// ShowEditor creates the window or raises it.
func (e *editorWindows) ShowEditor() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.w != nil {
		e.w.Perform(system.ActionRaise)
		e.ed.TakePending()
		return nil
	}
	w := new(app.Window)
	st := e.store.LoadUIState().EditorWindow
	width, height := st.Width, st.Height
	if width < 400 || height < 300 {
		width, height = 900, 640
	}
	// PureEdit's own identity (taskbar entry, icon), also in PureTerm's process.
	w.Option(append([]app.Option{app.Title("PureEdit"), app.Size(unit.Dp(width), unit.Dp(height)), app.MinSize(400, 300)},
		winsys.PureEdit.Options()...)...)
	ed := pureedit.New(e.svc)
	ed.Invalidate = w.Invalidate
	ed.SetTitle = func(s string) { w.Option(app.Title(s)) }
	ed.Shaper = e.shaper
	e.w, e.ed, e.closing = w, ed, false
	go e.run(w, ed)
	return nil
}

// CloseEditor closes the window without asking again.
func (e *editorWindows) CloseEditor() {
	e.mu.Lock()
	w := e.w
	e.closing = true
	e.mu.Unlock()
	if w != nil {
		w.Perform(system.ActionClose)
	}
}

// EditorShown reports whether the window is open.
func (e *editorWindows) EditorShown() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.w != nil
}

// ShowEditorPrefs opens PureEdit's settings window or raises it.
func (e *editorWindows) ShowEditorPrefs() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.prefsW != nil {
		e.prefsW.Perform(system.ActionRaise)
		return nil
	}
	w := new(app.Window)
	w.Option(append([]app.Option{app.Title(services.EditorAppName + " Settings"), app.Size(560, 620), app.MinSize(420, 360)},
		winsys.PureEdit.Options()...)...)
	p := &prefs.EditorPrefs{
		Load:  e.svc.GetEditorSettings,
		Save:  e.svc.SaveEditorSettings,
		Close: func() { w.Perform(system.ActionClose) },
	}
	p.F().ChooseFile = func(title, dir string, folder bool) (string, error) {
		return e.chooseFile(title, dir, folder)
	}
	p.F().Changed = w.Invalidate
	e.prefsW = w
	go func() {
		scheme := e.store.Config().General.ColorScheme
		th := gtk.New(scheme)
		var ops op.Ops
		for {
			switch ev := w.Event().(type) {
			case app.DestroyEvent:
				e.mu.Lock()
				e.prefsW = nil
				e.mu.Unlock()
				return
			case app.FrameEvent:
				if s := e.store.Config().General.ColorScheme; s != scheme {
					scheme = s
					th.SetScheme(s)
				}
				gtx := app.NewContext(&ops, ev)
				p.Layout(gtx, th)
				ev.Frame(gtx.Ops)
			case app.ViewEvent:
				winsys.View(winsys.PureEdit, ev)
			}
		}
	}()
	return nil
}

// CloseEditorPrefs closes the settings window.
func (e *editorWindows) CloseEditorPrefs() {
	e.mu.Lock()
	w := e.prefsW
	e.mu.Unlock()
	if w != nil {
		w.Perform(system.ActionClose)
	}
}

// activate brings the editor window to the front for the desktop's
// activation token of a process that handed its files over.
func (e *editorWindows) activate(token string) {
	e.mu.Lock()
	w := e.w
	e.mu.Unlock()
	if w != nil {
		w.Activate(token)
	}
}

// pending opens queued files (events.EditorPending).
func (e *editorWindows) pending() {
	e.mu.Lock()
	ed := e.ed
	e.mu.Unlock()
	if ed != nil {
		ed.TakePending()
	}
}

func (e *editorWindows) settingsChanged() {
	e.mu.Lock()
	ed, w := e.ed, e.w
	e.mu.Unlock()
	if ed != nil {
		ed.SettingsChanged()
		w.Invalidate()
	}
}

// shaper shapes the editor font (called on the editor's goroutine).
func (e *editorWindows) shaper(family string) *text.Shaper {
	if family == "" {
		return nil
	}
	if sh, ok := e.shapers[family]; ok {
		return sh
	}
	sh := familyShaper(family)
	if e.shapers == nil {
		e.shapers = map[string]*text.Shaper{}
	}
	e.shapers[family] = sh
	return sh
}

func (e *editorWindows) run(w *app.Window, ed *pureedit.App) {
	defer crashGuard(e.store.ConfigDir())
	scheme := e.store.Config().General.ColorScheme
	th := gtk.New(scheme)
	var ops op.Ops
	var size [2]int
	first := true
	var dropHWND uintptr
	for {
		switch ev := w.Event().(type) {
		case app.ViewEvent:
			winsys.View(winsys.PureEdit, ev)
			enableWindowsDrop(ev, &dropHWND, func(d dnd.Drop) { e.dropped(d.Paths) })
		case app.FileDropEvent:
			e.dropped(ev.Paths)
		case app.DestroyEvent:
			if ev.Err != nil {
				log.Print("PureEdit window: ", ev.Err)
			}
			st := e.store.LoadUIState()
			if size[0] > 0 {
				st.EditorWindow.Width, st.EditorWindow.Height = size[0], size[1]
				_ = e.store.SaveUIState(st)
			}
			e.mu.Lock()
			e.w, e.ed = nil, nil
			e.mu.Unlock()
			e.svc.WindowClosed()
			if e.standalone && e.onExit != nil {
				e.onExit()
			}
			return
		case *app.ClosingEvent:
			e.mu.Lock()
			closing := e.closing
			e.mu.Unlock()
			if !closing {
				// The editor asks about unsaved changes and closes itself.
				ev.Abort()
				ed.RequestClose()
				w.Invalidate()
			}
		case app.ConfigEvent:
			if ev.Config.Focused {
				ed.Focused()
			}
		case app.FrameEvent:
			if first {
				// A PureEdit the user just started comes to the front with
				// the desktop's token (else Wayland desktops may keep it behind).
				first = false
				if t := e.startToken; t != "" {
					e.startToken = ""
					go w.Activate(t)
				}
			}
			if s := e.store.Config().General.ColorScheme; s != scheme {
				scheme = s
				th.SetScheme(s)
			}
			gtx := app.NewContext(&ops, ev)
			size = [2]int{int(float32(ev.Size.X) / gtx.Metric.PxPerDp), int(float32(ev.Size.Y) / gtx.Metric.PxPerDp)}
			ed.Layout(gtx, th)
			ev.Frame(gtx.Ops)
		}
	}
}

// dropped opens files dropped on the editor window (folders are skipped).
func (e *editorWindows) dropped(paths []string) {
	go func() {
		for _, p := range paths {
			if fi, err := os.Stat(p); err != nil || fi.IsDir() {
				continue
			}
			if err := e.svc.Open(services.EditorFile{Kind: "local", Path: p}); err != nil {
				log.Print("PureEdit: ", err)
			}
		}
	}()
}

// watchEditorEvents connects the event bus to the editor window.
func (e *editorWindows) watchEditorEvents(bus *events.Bus) {
	bus.On(events.EditorPending, func(string, any) { e.pending() })
	bus.On(events.SettingsChanged, func(string, any) { e.settingsChanged() })
}

// pureEditHost is the services.Host of PureEdit on its own: file dialogs
// and quitting.
type pureEditHost struct{}

func (pureEditHost) ChooseFile(d services.FileDialog) (string, error) {
	return filedialog.Choose(filedialog.Options{Title: d.Title, Directory: d.Directory, Filename: d.Filename, Save: d.Save, Folder: d.Folder})
}
func (pureEditHost) Quit() { os.Exit(0) }

// runPureEdit starts only the editor (pureterm -pureedit [files], or the
// pureedit executable): no main window and no sessions; closing the window
// quits. Its settings go straight to config.toml (EditorService standalone).
//
// With [editor] single_instance (the default), files go to a PureEdit that
// is running already (see internal/singleinstance) and this process exits;
// the desktop's activation token goes along, so that window comes to the
// front. PureTerm's own editor window never takes part: a PureEdit started
// from the desktop is independent of PureTerm. newWindow (File → New Window)
// always opens a window of its own and leaves the lock to the running
// instance.
func runPureEdit(configDir string, args []string, newWindow bool) {
	winsys.SetProcess(winsys.PureEdit)
	token := winsys.StartupToken()
	files := editorArgs(args)
	store, err := config.NewStore(config.StoreOptions{ConfigDir: configDir, NoBackup: true})
	if err != nil {
		fatalStartup(configDir, err)
	}
	defer crashGuard(store.ConfigDir())
	// A configuration PureEdit cannot read must not keep it from editing
	// files: it runs with the defaults then (and does not save over it).
	if err := store.Load(); err != nil {
		logStartup(store.ConfigDir(), "PureEdit: the configuration could not be loaded, using the defaults: "+err.Error())
	}
	single := store.Config().Editor.SingleInstance && !newWindow
	if single {
		winsys.AllowActivation()
		forwarded, err := singleinstance.ForwardRequest(store.ConfigDir(), singleinstance.Request{Files: files, Activation: token})
		if err != nil {
			log.Print("PureEdit: ", err)
		}
		if forwarded {
			return
		}
	}
	bus := events.New()
	svc := services.NewEditorService(store, nil, bus)
	svc.SetStandalone(true)
	// Settings saved by PureTerm or another PureEdit apply right away.
	store.OnReload = func() { bus.Emit(events.SettingsChanged, nil) }
	host := pureEditHost{}
	var server *singleinstance.Server
	ew := &editorWindows{store: store, svc: svc, standalone: true, startToken: token, onExit: func() {
		if server != nil {
			_ = server.Close()
		}
		os.Exit(0)
	}, chooseFile: func(title, dir string, folder bool) (string, error) {
		return host.ChooseFile(services.FileDialog{Title: title, Directory: dir, Folder: folder})
	}}
	svc.SetGUI(host, ew)
	ew.watchEditorEvents(bus)
	if single {
		// Later instances hand their files over and exit (on the server's goroutine).
		server, err = singleinstance.ListenRequests(store.ConfigDir(), func(r singleinstance.Request) {
			if len(r.Files) == 0 {
				_ = svc.ShowWindow()
			}
			for _, p := range r.Files {
				_ = svc.Open(services.EditorFile{Kind: "local", Path: p})
			}
			ew.activate(r.Activation)
		})
		if err != nil {
			log.Print("PureEdit: single instance off: ", err)
			server = nil
		}
	}
	go func() {
		defer crashGuard(store.ConfigDir())
		opened := false
		for _, f := range files {
			if err := svc.Open(services.EditorFile{Kind: "local", Path: f}); err == nil {
				opened = true
			}
		}
		if !opened {
			if err := svc.ShowWindow(); err != nil {
				fatalStartup(store.ConfigDir(), err)
			}
		}
	}()
	app.Main()
}

// isPureEditExe reports whether this program was started as PureEdit's own
// executable (pureedit.exe on Windows, or a pureedit link).
func isPureEditExe() bool {
	name := strings.ToLower(filepath.Base(os.Args[0]))
	return strings.TrimSuffix(name, ".exe") == "pureedit"
}

// editorArgs turns PureEdit's arguments into absolute local paths. Desktops
// pass paths (%F) or file:// URIs (%U, percent-encoded); empty arguments are
// skipped, and relative paths are relative to the working folder.
func editorArgs(args []string) []string {
	var out []string
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if p, ok := fileURIPath(a); ok {
			a = p
		} else if strings.Contains(a, "://") {
			log.Printf("PureEdit: %s: only local files can be opened", a)
			continue
		}
		if abs, err := filepath.Abs(a); err == nil {
			a = abs
		}
		out = append(out, a)
	}
	return out
}

// fileURIPath returns the local path of a file:// URI.
func fileURIPath(s string) (string, bool) {
	if len(s) < 7 || !strings.EqualFold(s[:7], "file://") {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil || u.Path == "" {
		return "", false
	}
	if h := u.Host; h != "" && !strings.EqualFold(h, "localhost") {
		if name, _ := os.Hostname(); !strings.EqualFold(h, name) {
			return "", false
		}
	}
	p := u.Path
	// file:///C:/dir/file on Windows.
	if runtime.GOOS == "windows" && len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p), true
}

// startupLog is the file in the configuration folder where PureEdit and
// PureTerm note what kept them from starting (a desktop shows no terminal).
func startupLog(dir string) string { return filepath.Join(dir, "state", "startup.log") }

func logStartup(dir, msg string) {
	log.Print(msg)
	if dir == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(startupLog(dir)), 0o700)
	f, err := os.OpenFile(startupLog(dir), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
}

func fatalStartup(dir string, err error) {
	if dir == "" {
		dir = config.ResolveConfigDir("")
	}
	logStartup(dir, "start failed: "+err.Error())
	os.Exit(1)
}

// crashGuard notes a panic of the goroutine it is deferred in, with its
// stack, in the startup log before the program ends as it would.
func crashGuard(dir string) {
	if r := recover(); r != nil {
		logStartup(dir, fmt.Sprintf("panic: %v\n%s", r, debug.Stack()))
		panic(r)
	}
}
