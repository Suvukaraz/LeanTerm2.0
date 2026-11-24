//go:build windows

package fonts

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gioui.org/font"
	"golang.org/x/sys/windows/registry"
)

const windowsFontsKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts`

type regFontEntry struct {
	name string
	path string
}

var (
	regOnce    sync.Once
	regEntries []regFontEntry

	winIndexOnce sync.Once
	winIndex     map[string][]fontRef
)

func loadRegEntries() []regFontEntry {
	regOnce.Do(func() {
		fontDir := filepath.Join(os.Getenv("WINDIR"), "Fonts")
		seen := map[string]bool{}
		for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
			k, err := registry.OpenKey(root, windowsFontsKey, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			names, err := k.ReadValueNames(0)
			if err == nil {
				for _, n := range names {
					v, _, err := k.GetStringValue(n)
					if err != nil || v == "" {
						continue
					}
					if !filepath.IsAbs(v) {
						v = filepath.Join(fontDir, v)
					}
					switch strings.ToLower(filepath.Ext(v)) {
					case ".ttf", ".otf", ".ttc", ".otc":
					default:
						continue
					}
					key := strings.ToLower(v)
					if seen[key] {
						continue
					}
					seen[key] = true
					regEntries = append(regEntries, regFontEntry{name: n, path: v})
				}
			}
			_ = k.Close()
		}
	})
	return regEntries
}

// regNameMatchesFamily checks whether a Windows Fonts registry value name
// (e.g. "Consolas Bold (TrueType)" or "MS Gothic & MS UI Gothic (TrueType)")
// plausibly belongs to family, so the fast path only opens 1-4 files.
func regNameMatchesFamily(regName, family string) bool {
	if idx := strings.LastIndex(regName, "("); idx > 0 {
		regName = regName[:idx]
	}
	for _, part := range strings.Split(regName, "&") {
		p := strings.TrimSpace(part)
		if strings.EqualFold(p, family) {
			return true
		}
		if len(p) > len(family) && strings.EqualFold(p[:len(family)], family) && p[len(family)] == ' ' {
			return true
		}
	}
	return false
}

func findFace(family, style string) (font.FontFace, bool) {
	entries := loadRegEntries()
	var refs []fontRef
	for _, e := range entries {
		if !regNameMatchesFamily(e.name, family) {
			continue
		}
		f, err := os.Open(e.path)
		if err != nil {
			continue
		}
		names := readFontFamilies(f)
		_ = f.Close()
		for _, idx := range entriesMatchFamily(names, family) {
			refs = append(refs, fontRef{Path: e.path, Index: idx})
		}
	}
	if len(refs) == 0 {
		winIndexOnce.Do(func() {
			var paths []string
			seen := map[string]bool{}
			for _, e := range entries {
				k := strings.ToLower(e.path)
				if !seen[k] {
					seen[k] = true
					paths = append(paths, e.path)
				}
			}
			var dirs []string
			if wd := os.Getenv("WINDIR"); wd != "" {
				dirs = append(dirs, filepath.Join(wd, "Fonts"))
			}
			if la := os.Getenv("LOCALAPPDATA"); la != "" {
				dirs = append(dirs, filepath.Join(la, "Microsoft", "Windows", "Fonts"))
			}
			for _, p := range collectFontPaths(dirs) {
				k := strings.ToLower(p)
				if !seen[k] {
					seen[k] = true
					paths = append(paths, p)
				}
			}
			winIndex = indexFontFiles(paths)
		})
		refs = winIndex[strings.ToLower(strings.TrimSpace(family))]
	}
	if len(refs) == 0 {
		return font.FontFace{}, false
	}
	matches := loadFontRefs(refs)
	if len(matches) == 0 {
		return font.FontFace{}, false
	}
	return pick(matches, style), true
}
