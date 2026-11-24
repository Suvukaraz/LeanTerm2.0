//go:build !windows

package fonts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

var (
	otherIndexOnce sync.Once
	otherIndex     map[string][]fontRef
)

func findFace(family, style string) (font.FontFace, bool) {
	if runtime.GOOS == "linux" {
		if fc, err := exec.LookPath("fc-match"); err == nil {
			pattern := family
			if style != "" {
				pattern += ":style=" + style
			}
			out, err := exec.Command(fc, "-f", "%{family}\n%{file}\n%{index}", pattern).Output()
			lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 3)
			// fontconfig substitutes another family for an unknown one.
			if err == nil && len(lines) >= 2 && familyMatches(lines[0], family) {
				if data, err := os.ReadFile(lines[1]); err == nil {
					idx := 0
					if len(lines) == 3 {
						idx, _ = strconv.Atoi(strings.TrimSpace(lines[2]))
					}
					if f, err := opentype.ParseIndex(data, idx); err == nil {
						return f, true
					}
				}
			}
			return font.FontFace{}, false
		}
	}
	otherIndexOnce.Do(func() {
		home, _ := os.UserHomeDir()
		dirs := []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			filepath.Join(home, ".fonts"),
			filepath.Join(home, ".local", "share", "fonts"),
		}
		otherIndex = indexFontFiles(collectFontPaths(dirs))
	})
	refs := otherIndex[strings.ToLower(strings.TrimSpace(family))]
	if len(refs) == 0 {
		return font.FontFace{}, false
	}
	matches := loadFontRefs(refs)
	if len(matches) == 0 {
		return font.FontFace{}, false
	}
	return pick(matches, style), true
}
