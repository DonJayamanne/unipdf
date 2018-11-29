/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.md', which is part of this source code package.
 */

package textencoding

import (
	"fmt"
	"sort"
	"strings"

	"github.com/unidoc/unidoc/common"
	"github.com/unidoc/unidoc/pdf/core"
)

type GID uint16

// TrueTypeFontEncoder handles text encoding for composite TrueType fonts.
// It performs mapping between character ids and glyph ids.
// It has a preloaded rune (unicode code point) to glyph index map that has been loaded from a font.
// Corresponds to Identity-H.
type TrueTypeFontEncoder struct {
	runeToGlyphIndexMap map[rune]GID
	cmap                CMap
}

// NewTrueTypeFontEncoder creates a new text encoder for TTF fonts with a pre-loaded
// runeToGlyphIndexMap, that has been pre-loaded from the font file.
// The new instance is preloaded with a CMapIdentityH (Identity-H) CMap which maps 2-byte charcodes
// to CIDs (glyph index).
func NewTrueTypeFontEncoder(runeToGlyphIndexMap map[rune]GID) TrueTypeFontEncoder {
	return TrueTypeFontEncoder{
		runeToGlyphIndexMap: runeToGlyphIndexMap,
		cmap:                CMapIdentityH{},
	}
}

// ttEncoderMaxNumEntries is the maximum number of encoding entries shown in SimpleEncoder.String().
const ttEncoderMaxNumEntries = 10

// String returns a string that describes `enc`.
func (enc TrueTypeFontEncoder) String() string {
	parts := []string{
		fmt.Sprintf("%d entries", len(enc.runeToGlyphIndexMap)),
	}

	runes := make([]rune, 0, len(enc.runeToGlyphIndexMap))
	for r := range enc.runeToGlyphIndexMap {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	n := len(runes)
	if n > ttEncoderMaxNumEntries {
		n = ttEncoderMaxNumEntries
	}

	for i := 0; i < n; i++ {
		r := runes[i]
		parts = append(parts, fmt.Sprintf("%d=0x%02x: %q",
			r, r, enc.runeToGlyphIndexMap[r]))
	}
	return fmt.Sprintf("TRUETYPE_ENCODER{%s}", strings.Join(parts, ", "))
}

// Encode converts the Go unicode string `raw` to a PDF encoded string.
func (enc TrueTypeFontEncoder) Encode(raw string) []byte {
	return encodeString16bit(enc, raw)
}

// CharcodeToGlyph returns the glyph name matching character code `code`.
// The bool return flag is true if there was a match, and false otherwise.
func (enc TrueTypeFontEncoder) CharcodeToGlyph(code CharCode) (string, bool) {
	r, found := enc.CharcodeToRune(code)
	if found && r == 0x20 {
		return "space", true
	}

	// Returns "uniXXXX" format where XXXX is the code in hex format.
	glyph := fmt.Sprintf("uni%.4X", code)
	return glyph, true
}

// GlyphToCharcode returns character code matching the glyph name `glyph`.
// The bool return flag is true if there was a match, and false otherwise.
func (enc TrueTypeFontEncoder) GlyphToCharcode(glyph string) (CharCode, bool) {
	// String with "uniXXXX" format where XXXX is the hexcode.
	if len(glyph) == 7 && glyph[0:3] == "uni" {
		var unicode uint16
		n, err := fmt.Sscanf(glyph, "uni%X", &unicode)
		if n == 1 && err == nil {
			return enc.RuneToCharcode(rune(unicode))
		}
	}

	// Look in glyphlist.
	if rune, found := glyphlistGlyphToRuneMap[glyph]; found {
		return enc.RuneToCharcode(rune)
	}

	common.Log.Debug("Symbol encoding error: unable to find glyph->charcode entry (%s)", glyph)
	return 0, false
}

// RuneToCharcode converts rune `r` to a PDF character code.
// The bool return flag is true if there was a match, and false otherwise.
func (enc TrueTypeFontEncoder) RuneToCharcode(r rune) (CharCode, bool) {
	glyphIndex, ok := enc.runeToGlyphIndexMap[r]
	if !ok {
		common.Log.Debug("Missing rune %d (%+q) from encoding", r, r)
		return 0, false
	}
	// Identity : charcode <-> glyphIndex
	charcode := CharCode(glyphIndex)

	return charcode, true
}

// CharcodeToRune converts PDF character code `code` to a rune.
// The bool return flag is true if there was a match, and false otherwise.
func (enc TrueTypeFontEncoder) CharcodeToRune(code CharCode) (rune, bool) {
	// TODO: Make a reverse map stored.
	for r, glyphIndex := range enc.runeToGlyphIndexMap {
		// Identity : glyphIndex <-> charcode
		charcode := CharCode(glyphIndex)
		if charcode == code {
			return r, true
		}
	}
	common.Log.Debug("CharcodeToRune: No match. code=0x%04x enc=%s", code, enc)
	return 0, false
}

// RuneToGlyph returns the glyph name for rune `r`.
// The bool return flag is true if there was a match, and false otherwise.
func (enc TrueTypeFontEncoder) RuneToGlyph(r rune) (string, bool) {
	if r == 0x20 {
		return "space", true
	}
	glyph := fmt.Sprintf("uni%.4X", r)
	return glyph, true
}

// GlyphToRune returns the rune corresponding to glyph name `glyph`.
// The bool return flag is true if there was a match, and false otherwise.
func (enc TrueTypeFontEncoder) GlyphToRune(glyph string) (rune, bool) {
	// String with "uniXXXX" format where XXXX is the hexcode.
	if len(glyph) == 7 && glyph[0:3] == "uni" {
		unicode := uint16(0)
		n, err := fmt.Sscanf(glyph, "uni%X", &unicode)
		if n == 1 && err == nil {
			return rune(unicode), true
		}
	}

	// Look in glyphlist.
	if r, ok := glyphlistGlyphToRuneMap[glyph]; ok {
		return r, true
	}

	return 0, false
}

// ToPdfObject returns a nil as it is not truly a PDF object and should not be attempted to store in file.
func (enc TrueTypeFontEncoder) ToPdfObject() core.PdfObject {
	return core.MakeNull()
}
