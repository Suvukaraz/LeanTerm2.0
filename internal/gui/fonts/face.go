package fonts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"gioui.org/font"
)

var (
	faceMu    sync.Mutex
	faceCache = map[string]*font.FontFace{}
)

// Face loads the face of an installed font family in a style ("Bold",
// "Italic", ...), e.g. for previews in the font chooser. On Linux fontconfig
// finds the file; on Windows the font registry and lightweight 'name' table
// indexing find only the matching files.
func Face(family, style string) (font.FontFace, bool) {
	if isGenericFamily(family) {
		return font.FontFace{}, false
	}
	key := family + "\x00" + style
	faceMu.Lock()
	if f, ok := faceCache[key]; ok {
		faceMu.Unlock()
		if f == nil {
			return font.FontFace{}, false
		}
		return *f, true
	}
	faceMu.Unlock()

	f, ok := findFace(family, style)
	faceMu.Lock()
	if ok {
		faceCache[key] = &f
	} else {
		faceCache[key] = nil
	}
	faceMu.Unlock()
	return f, ok
}

func isGenericFamily(family string) bool {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "", "mono", "monospace", "sans", "sans-serif", "system-ui", "serif":
		return true
	}
	return false
}

// familyMatches compares a family with fontconfig's list of names for it.
func familyMatches(names, family string) bool {
	for _, n := range strings.Split(names, ",") {
		if strings.EqualFold(strings.TrimSpace(n), family) {
			return true
		}
	}
	return false
}

// pick chooses the face for a style name by weight and slant.
func pick(faces []font.FontFace, style string) font.FontFace {
	s := strings.ToLower(style)
	want := font.Font{Weight: font.Normal}
	switch {
	case strings.Contains(s, "black") || strings.Contains(s, "heavy"):
		want.Weight = font.Black
	case strings.Contains(s, "extrabold") || strings.Contains(s, "ultrabold"):
		want.Weight = font.ExtraBold
	case strings.Contains(s, "semibold") || strings.Contains(s, "demibold"):
		want.Weight = font.SemiBold
	case strings.Contains(s, "bold"):
		want.Weight = font.Bold
	case strings.Contains(s, "medium"):
		want.Weight = font.Medium
	case strings.Contains(s, "light"):
		want.Weight = font.Light
	case strings.Contains(s, "thin"):
		want.Weight = font.Thin
	}
	if strings.Contains(s, "italic") || strings.Contains(s, "oblique") {
		want.Style = font.Italic
	}
	best, bestScore := faces[0], -1
	for _, f := range faces {
		score := 0
		if f.Font.Style == want.Style {
			score += 1000
		}
		d := int(f.Font.Weight - want.Weight)
		if d < 0 {
			d = -d
		}
		score -= d
		if bestScore < 0 || score > bestScore {
			best, bestScore = f, score
		}
	}
	return best
}

// indexFontFiles reads only SFNT 'name' tables in parallel across the given
// files, returning a lowercase-family -> []fontRef lookup map.
func indexFontFiles(paths []string) map[string][]fontRef {
	index := map[string][]fontRef{}
	if len(paths) == 0 {
		return index
	}
	workers := min(max(runtime.GOMAXPROCS(0), 2), 8)
	ch := make(chan string, len(paths))
	for _, p := range paths {
		ch <- p
	}
	close(ch)

	type item struct {
		family string
		ref    fontRef
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local []item
			for p := range ch {
				f, err := os.Open(p)
				if err != nil {
					continue
				}
				entries := readFontFamilies(f)
				_ = f.Close()
				for _, e := range entries {
					for _, fam := range e.Families {
						local = append(local, item{
							family: strings.ToLower(fam),
							ref:    fontRef{Path: p, Index: e.Index},
						})
					}
				}
			}
			if len(local) == 0 {
				return
			}
			mu.Lock()
			for _, it := range local {
				index[it.family] = append(index[it.family], it.ref)
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return index
}

func collectFontPaths(dirs []string) []string {
	var paths []string
	seen := map[string]bool{}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		_ = filepath.WalkDir(d, func(path string, e os.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".ttf", ".otf", ".ttc", ".otc":
			default:
				return nil
			}
			key := strings.ToLower(path)
			if !seen[key] {
				seen[key] = true
				paths = append(paths, path)
			}
			return nil
		})
	}
	return paths
}
