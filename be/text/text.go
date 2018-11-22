package text

import (
	"unicode"

	"golang.org/x/text/unicode/rangetable"

	"pixur.org/pixur/be/status"
)

const UnicodeVersion = unicode.Version

func buildCc() (allowed, notallowed *unicode.RangeTable) {
	ccallowed := &unicode.RangeTable{
		R16: []unicode.Range16{
			{Lo: 0x09, Hi: 0x0a, Stride: 1}, // \t, \r
			{Lo: 0x0d, Hi: 0x0d, Stride: 1}, // \n
		},
		LatinOffset: 2,
	}
	// fast path
	if len(unicode.Cc.R16) == 2 && len(unicode.Cc.R32) == 0 && unicode.Cc.LatinOffset == 2 {
		if unicode.Cc.R16[0] == (unicode.Range16{Lo: 0, Hi: 0x1f, Stride: 1}) &&
			unicode.Cc.R16[1] == (unicode.Range16{Lo: 0x7f, Hi: 0x9f, Stride: 1}) {
			return ccallowed, &unicode.RangeTable{
				R16: []unicode.Range16{
					{Lo: 0, Hi: 0x08, Stride: 1},
					{Lo: 0x0b, Hi: 0x0c, Stride: 1},
					{Lo: 0x0e, Hi: 0x1f, Stride: 1},
					{Lo: 0x7f, Hi: 0x9f, Stride: 1},
				},
				LatinOffset: 4,
			}
		}
	}

	// TODO: implement
	panic("unexpected cc")
}

var notAllowedRange *unicode.RangeTable

func init() {
	_, ncc := buildCc()

	notAllowedRange = rangetable.Merge(ncc, unicode.Co, unicode.Cs, unicode.Noncharacter_Code_Point)
	// Forbidden:
	// unicode.Noncharacter_Code_Point

}

func ValidateCharset(s string) error {
	return validateCharset(s)
}

func validateCharset(s string) status.S {
	return nil
}

/*
#      Lu + Ll + Lt + Lm + Lo + Nl
#    + Other_ID_Start
#    - Pattern_Syntax
#    - Pattern_White_Space
*/
