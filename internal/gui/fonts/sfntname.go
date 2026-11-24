package fonts

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
	"unicode/utf16"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

type fontNameEntry struct {
	Index    int
	Families []string
}

type fontRef struct {
	Path  string
	Index int
}

// readFontFamilies reads only the table directory and 'name' table of a
// .ttf, .otf or .ttc file so indexing hundreds of fonts never reads full files
// or parses glyph/shaping tables.
func readFontFamilies(r io.ReaderAt) []fontNameEntry {
	var tag [4]byte
	if _, err := r.ReadAt(tag[:], 0); err != nil {
		return nil
	}
	if string(tag[:]) != "ttcf" {
		if fams := readFaceFamilies(r, 0); len(fams) > 0 {
			return []fontNameEntry{{Index: 0, Families: fams}}
		}
		return nil
	}
	var hdr [12]byte
	if _, err := r.ReadAt(hdr[:], 0); err != nil {
		return nil
	}
	n := binary.BigEndian.Uint32(hdr[8:])
	if n == 0 || n > 256 {
		return nil
	}
	offs := make([]byte, 4*n)
	if _, err := r.ReadAt(offs, 12); err != nil {
		return nil
	}
	var out []fontNameEntry
	for i := uint32(0); i < n; i++ {
		if fams := readFaceFamilies(r, int64(binary.BigEndian.Uint32(offs[4*i:]))); len(fams) > 0 {
			out = append(out, fontNameEntry{Index: int(i), Families: fams})
		}
	}
	return out
}

func readFaceFamilies(r io.ReaderAt, at int64) []string {
	var hdr [12]byte
	if _, err := r.ReadAt(hdr[:], at); err != nil {
		return nil
	}
	switch binary.BigEndian.Uint32(hdr[:4]) {
	case 0x00010000, 0x4F54544F /* OTTO */, 0x74727565 /* true */ :
	default:
		return nil
	}
	numTables := int(binary.BigEndian.Uint16(hdr[4:]))
	if numTables == 0 || numTables > 512 {
		return nil
	}
	dir := make([]byte, 16*numTables)
	if _, err := r.ReadAt(dir, at+12); err != nil {
		return nil
	}
	for i := 0; i < numTables; i++ {
		e := dir[16*i:]
		if string(e[:4]) != "name" {
			continue
		}
		off := binary.BigEndian.Uint32(e[8:])
		length := binary.BigEndian.Uint32(e[12:])
		if length == 0 || length >= 1<<20 {
			return nil
		}
		buf := make([]byte, length)
		if _, err := r.ReadAt(buf, int64(off)); err != nil {
			return nil
		}
		return parseNameFamilies(buf)
	}
	return nil
}

// parseNameFamilies returns distinct family names (name IDs 16 and 1) from
// an OpenType 'name' table.
func parseNameFamilies(b []byte) []string {
	if len(b) < 6 {
		return nil
	}
	count := int(binary.BigEndian.Uint16(b[2:]))
	strOff := int(binary.BigEndian.Uint16(b[4:]))
	best16, best1 := "", ""
	rank16, rank1 := 0, 0
	for i := 0; i < count; i++ {
		rec := 6 + 12*i
		if rec+12 > len(b) {
			break
		}
		platform := binary.BigEndian.Uint16(b[rec:])
		encoding := binary.BigEndian.Uint16(b[rec+2:])
		lang := binary.BigEndian.Uint16(b[rec+4:])
		id := int(binary.BigEndian.Uint16(b[rec+6:]))
		length := int(binary.BigEndian.Uint16(b[rec+8:]))
		off := strOff + int(binary.BigEndian.Uint16(b[rec+10:]))
		if (id != 1 && id != 16) || off+length > len(b) {
			continue
		}
		raw := b[off : off+length]
		var s string
		r := 0
		switch {
		case platform == 3 && (encoding == 1 || encoding == 10):
			u := make([]uint16, len(raw)/2)
			for j := range u {
				u[j] = binary.BigEndian.Uint16(raw[2*j:])
			}
			s = string(utf16.Decode(u))
			r = 2
			if lang == 0x0409 {
				r = 3
			}
		case platform == 1 && encoding == 0:
			s = string(raw)
			r = 1
		default:
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if id == 16 && r > rank16 {
			best16, rank16 = s, r
		} else if id == 1 && r > rank1 {
			best1, rank1 = s, r
		}
	}
	var out []string
	if best16 != "" {
		out = append(out, best16)
	}
	if best1 != "" && !strings.EqualFold(best1, best16) {
		out = append(out, best1)
	}
	return out
}

func entriesMatchFamily(entries []fontNameEntry, family string) []int {
	var indices []int
	for _, e := range entries {
		for _, f := range e.Families {
			if strings.EqualFold(f, family) {
				indices = append(indices, e.Index)
				break
			}
		}
	}
	return indices
}

// loadFontRefs parses only the listed (path, index) faces, reading each file once.
func loadFontRefs(refs []fontRef) []font.FontFace {
	byPath := map[string][]int{}
	var paths []string
	for _, r := range refs {
		if _, ok := byPath[r.Path]; !ok {
			paths = append(paths, r.Path)
		}
		byPath[r.Path] = append(byPath[r.Path], r.Index)
	}
	var out []font.FontFace
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, idx := range byPath[p] {
			if ff, err := opentype.ParseIndex(data, idx); err == nil {
				out = append(out, ff)
			}
		}
	}
	return out
}
