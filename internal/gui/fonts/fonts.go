// Package fonts loads the faces PureTerm's GUI shapes text with: the system's
// UI and monospace fonts when they can be found, else Gio's Go fonts.
package fonts

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"golang.org/x/image/font/gofont/gomono"
)

// Typefaces used by the GUI.
const (
	UI        font.Typeface = "PureTerm UI"
	Monospace font.Typeface = "PureTerm Mono"
	// Symbols is the Nerd Fonts symbol font, for icons (see icons.go).
	Symbols font.Typeface = "Symbols Nerd Font"
)

//go:embed nerd/SymbolsNerdFont-Regular.ttf
var symbolsTTF []byte

type candidate struct {
	typeface font.Typeface
	style    font.Style
	weight   font.Weight
	paths    []string
}

// fontconfigCandidates asks fontconfig for the faces the view would use
// (system-ui, monospace), so both GUIs draw with the same fonts.
func fontconfigCandidates() []candidate {
	if runtime.GOOS != "linux" {
		return nil
	}
	fcMatch, err := exec.LookPath("fc-match")
	if err != nil {
		return nil
	}
	type query struct {
		typeface font.Typeface
		pattern  string
		style    font.Style
		weight   font.Weight
	}
	variants := []struct {
		suffix string
		style  font.Style
		weight font.Weight
	}{
		{"", font.Regular, font.Normal},
		{":medium", font.Regular, font.Medium},
		{":semibold", font.Regular, font.SemiBold},
		{":bold", font.Regular, font.Bold},
		{":italic", font.Italic, font.Normal},
		{":bold:italic", font.Italic, font.Bold},
	}
	var queries []query
	for _, tf := range []struct {
		typeface font.Typeface
		pattern  string
	}{{UI, "system-ui"}, {Monospace, "monospace"}} {
		for _, v := range variants {
			queries = append(queries, query{tf.typeface, tf.pattern + v.suffix, v.style, v.weight})
		}
	}
	paths := make([]string, len(queries))
	var wg sync.WaitGroup
	for i, q := range queries {
		wg.Add(1)
		go func(i int, pat string) {
			defer wg.Done()
			if b, err := exec.Command(fcMatch, "-f", "%{file}", pat).Output(); err == nil {
				paths[i] = strings.TrimSpace(string(b))
			}
		}(i, q.pattern)
	}
	wg.Wait()

	var out []candidate
	seen := map[string]bool{}
	for i, q := range queries {
		p := paths[i]
		// A weight the family lacks resolves to another face: skip it, the
		// shaper picks the nearest one.
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, candidate{q.typeface, q.style, q.weight, []string{p}})
	}
	return out
}

func candidates() []candidate {
	if c := fontconfigCandidates(); len(c) > 0 {
		return c
	}
	if runtime.GOOS == "windows" {
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		f := func(names ...string) []string {
			out := make([]string, len(names))
			for i, n := range names {
				out[i] = filepath.Join(dir, n)
			}
			return out
		}
		return []candidate{
			{UI, font.Regular, font.Normal, f("segoeui.ttf")},
			{UI, font.Regular, font.SemiBold, f("seguisb.ttf")},
			{UI, font.Regular, font.Bold, f("segoeuib.ttf")},
			{UI, font.Italic, font.Normal, f("segoeuii.ttf")},
			{Monospace, font.Regular, font.Normal, f("CascadiaMono.ttf", "consola.ttf")},
			{Monospace, font.Regular, font.Bold, f("consolab.ttf")},
			{Monospace, font.Italic, font.Normal, f("consolai.ttf")},
			{Monospace, font.Italic, font.Bold, f("consolaz.ttf")},
		}
	}
	dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
	f := func(rel ...string) []string {
		var out []string
		for _, d := range dirs {
			for _, r := range rel {
				out = append(out, filepath.Join(d, r))
			}
		}
		return out
	}
	return []candidate{
		{UI, font.Regular, font.Normal, f("truetype/noto/NotoSans-Regular.ttf", "noto/NotoSans-Regular.ttf", "google-noto/NotoSans-Regular.ttf", "truetype/dejavu/DejaVuSans.ttf", "dejavu/DejaVuSans.ttf", "TTF/DejaVuSans.ttf")},
		{UI, font.Regular, font.Bold, f("truetype/noto/NotoSans-Bold.ttf", "noto/NotoSans-Bold.ttf", "google-noto/NotoSans-Bold.ttf", "truetype/dejavu/DejaVuSans-Bold.ttf", "dejavu/DejaVuSans-Bold.ttf", "TTF/DejaVuSans-Bold.ttf")},
		{Monospace, font.Regular, font.Normal, f("truetype/hack/Hack-Regular.ttf", "truetype/dejavu/DejaVuSansMono.ttf", "dejavu/DejaVuSansMono.ttf", "TTF/DejaVuSansMono.ttf", "truetype/noto/NotoSansMono-Regular.ttf")},
		{Monospace, font.Regular, font.Bold, f("truetype/hack/Hack-Bold.ttf", "truetype/dejavu/DejaVuSansMono-Bold.ttf", "dejavu/DejaVuSansMono-Bold.ttf", "TTF/DejaVuSansMono-Bold.ttf", "truetype/noto/NotoSansMono-Bold.ttf")},
		{Monospace, font.Italic, font.Normal, f("truetype/hack/Hack-Italic.ttf", "truetype/dejavu/DejaVuSansMono-Oblique.ttf", "dejavu/DejaVuSansMono-Oblique.ttf", "TTF/DejaVuSansMono-Oblique.ttf")},
		{Monospace, font.Italic, font.Bold, f("truetype/hack/Hack-BoldItalic.ttf", "truetype/dejavu/DejaVuSansMono-BoldOblique.ttf", "dejavu/DejaVuSansMono-BoldOblique.ttf", "TTF/DejaVuSansMono-BoldOblique.ttf")},
	}
}

var (
	once       sync.Once
	collection []font.FontFace
)

// monoData is the file of the regular monospace face.
var monoData []byte

// MonospaceData returns the font file of the regular monospace face (Go
// Mono without a system one), e.g. to draw terminal screenshots.
func MonospaceData() []byte {
	Collection()
	if monoData != nil {
		return monoData
	}
	return gomono.TTF
}

// Collection returns the system faces found (registered under [UI] and
// [Monospace]) followed by the Go fonts as fallback, also for glyphs the
// system faces lack.
func Collection() []font.FontFace {
	once.Do(func() {
		var (
			cands     []candidate
			fbSrcs    []fallbackSrc
			candFaces []font.FontFace
			candMono  []byte
			fbFaces   []font.FontFace
			symFace   opentype.Face
			symOK     bool
		)

		// Discover candidates and fallback paths concurrently (fc-match on Linux).
		var discWG sync.WaitGroup
		discWG.Add(2)
		go func() {
			defer discWG.Done()
			cands = candidates()
		}()
		go func() {
			defer discWG.Done()
			fbSrcs = fallbackSources()
		}()

		// Parse the embedded symbols font concurrently with discovery.
		var parseWG sync.WaitGroup
		parseWG.Add(1)
		go func() {
			defer parseWG.Done()
			if f, err := opentype.Parse(symbolsTTF); err == nil {
				symFace, symOK = f, true
			}
		}()

		discWG.Wait()

		candFaces = make([]font.FontFace, len(cands))
		candOK := make([]bool, len(cands))
		for i, c := range cands {
			parseWG.Add(1)
			go func(i int, c candidate) {
				defer parseWG.Done()
				for _, p := range c.paths {
					data, err := os.ReadFile(p)
					if err != nil {
						continue
					}
					face, err := opentype.Parse(data)
					if err != nil {
						continue
					}
					candFaces[i] = font.FontFace{
						Font: font.Font{Typeface: c.typeface, Style: c.style, Weight: c.weight},
						Face: face,
					}
					candOK[i] = true
					if c.typeface == Monospace && c.style == font.Regular && c.weight == font.Normal {
						candMono = data
					}
					return
				}
			}(i, c)
		}

		// Read each unique fallback file once, then parse each needed face index in parallel.
		if len(fbSrcs) > 0 {
			parseWG.Add(1)
			go func() {
				defer parseWG.Done()
				fbFaces = loadFallbackFaces(fbSrcs)
			}()
		}

		parseWG.Wait()

		for i := range cands {
			if candOK[i] {
				collection = append(collection, candFaces[i])
			}
		}
		if candMono != nil {
			monoData = candMono
		}
		// System fonts for scripts and emoji the UI and monospace fonts
		// lack (CJK, color emoji): the shaper falls back to them.
		collection = append(collection, fbFaces...)
		// Nerd Font symbols: icons, and the fallback for Powerline and other
		// Nerd Font glyphs in prompts.
		if symOK {
			collection = append(collection, font.FontFace{Font: font.Font{Typeface: Symbols}, Face: symFace})
		}
		// Go fonts under our names, so a missing system face still resolves.
		// Only where there is no system face: with two faces of the same
		// name, weight and style Gio may pick the Go one.
		have := map[font.Font]bool{}
		for _, ff := range collection {
			have[ff.Font] = true
		}
		for _, ff := range gofont.Collection() {
			alias := ff
			switch ff.Font.Typeface {
			case "Go Mono":
				alias.Font.Typeface = Monospace
			case "Go":
				alias.Font.Typeface = UI
			default: // Go Smallcaps, Go Medium etc. are not aliased
				collection = append(collection, ff)
				continue
			}
			collection = append(collection, ff)
			if !have[alias.Font] {
				collection = append(collection, alias)
			}
		}
	})
	return collection
}

// Fallback is the name prefix of the system faces for characters the UI
// and monospace fonts do not have ("PureTerm Fallback 1", ...): each face
// needs its own name, as Gio keeps one face per name, weight and style.
const Fallback font.Typeface = "PureTerm Fallback"

type fallbackSrc struct {
	path  string
	index int
}

// fallbackSources finds system fonts for CJK and emoji: fontconfig's choice
// on Linux, the fonts Windows ships elsewhere.
func fallbackSources() []fallbackSrc {
	var srcs []fallbackSrc
	seen := map[fallbackSrc]bool{}
	add := func(s fallbackSrc) {
		if s.path != "" && !seen[s] {
			seen[s] = true
			srcs = append(srcs, s)
		}
	}
	if runtime.GOOS == "linux" {
		if fc, err := exec.LookPath("fc-match"); err == nil {
			patterns := []string{"monospace:lang=ja", "monospace:lang=zh-cn", "monospace:lang=ko", "emoji"}
			found := make([]fallbackSrc, len(patterns))
			var wg sync.WaitGroup
			for i, pattern := range patterns {
				wg.Add(1)
				go func(i int, pat string) {
					defer wg.Done()
					out, err := exec.Command(fc, "-f", "%{file}\n%{index}", pat).Output()
					if err != nil {
						return
					}
					parts := strings.SplitN(string(out), "\n", 2)
					idx := 0
					if len(parts) == 2 {
						idx, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
					}
					found[i] = fallbackSrc{strings.TrimSpace(parts[0]), idx}
				}(i, pattern)
			}
			wg.Wait()
			for _, s := range found {
				add(s)
			}
		}
	}
	if runtime.GOOS == "windows" {
		dir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		for _, n := range []string{"msgothic.ttc", "msyh.ttc", "malgun.ttf", "seguiemj.ttf"} {
			add(fallbackSrc{filepath.Join(dir, n), 0})
		}
	}
	return srcs
}

func loadFallbackFaces(srcs []fallbackSrc) []font.FontFace {
	// Read each unique file once (on Linux, CJK patterns often share one .ttc).
	uniquePaths := make([]string, 0, len(srcs))
	fileData := map[string][]byte{}
	for _, s := range srcs {
		if _, ok := fileData[s.path]; !ok {
			fileData[s.path] = nil
			uniquePaths = append(uniquePaths, s.path)
		}
	}
	var readWG sync.WaitGroup
	var mu sync.Mutex
	for _, p := range uniquePaths {
		readWG.Add(1)
		go func(p string) {
			defer readWG.Done()
			if data, err := os.ReadFile(p); err == nil {
				mu.Lock()
				fileData[p] = data
				mu.Unlock()
			}
		}(p)
	}
	readWG.Wait()

	parsed := make([]font.FontFace, len(srcs))
	ok := make([]bool, len(srcs))
	var parseWG sync.WaitGroup
	for i, s := range srcs {
		data := fileData[s.path]
		if len(data) == 0 {
			continue
		}
		parseWG.Add(1)
		go func(i int, s fallbackSrc, data []byte) {
			defer parseWG.Done()
			f, err := opentype.ParseIndex(data, s.index)
			if err != nil {
				return
			}
			parsed[i] = f
			ok[i] = true
		}(i, s, data)
	}
	parseWG.Wait()

	var out []font.FontFace
	for i := range srcs {
		if !ok[i] {
			continue
		}
		f := parsed[i]
		f.Font = font.Font{Typeface: font.Typeface(fmt.Sprintf("%s %d", Fallback, len(out)+1))}
		out = append(out, f)
	}
	return out
}
