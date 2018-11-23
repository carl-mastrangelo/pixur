package text

import (
	"unicode"
	"unicode/utf8"

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
}

// ValidateEncoding ensures that the given string is valid UTF-8.
func ValidateEncoding(s string) error {
	return validateEncoding(s)
}

func validateEncoding(s string) status.S {
	if !utf8.ValidString(s) {
		return status.InvalidArgument(nil, "bad utf8 encoding")
	}
	return nil
}

// ValidateCodepoints ensures that the string only contains reasonable characters.  By default,
// Unicode classes Cc (controls), Co (private use), Cs (surrogates), and Noncharacters are
// forbidden.  (\t, \r, and \n are allowed, though).   Allowed classes include L (letters), M
// (marks), N (numbers), P (punctuation), S (symbols), Z (separators/spaces), and Cf (formats).
// Additionally, reserved characters, i.e. those for future use are allowed, as they may become
// mapped in the future.
func ValidateCodepoints(s string) error {
	return validateCodepoints(s)
}

// validateCodepoints does a blacklist check rather than a whitelist.  Since the allowed and not
// allowed sets are finite and exclusive, it is equivalent to a whitelist check.  It's possible
// that the unicode spec could move some characters from the allowed range into the not allowed
// range (i.e. from reserved to control), but that's less bad than rejecting future valid sequences
// (i.e. reserved to letter).
func validateCodepoints(s string) status.S {
	for i, r := range s {
		// Incase s hasn't been checked with validateEncoding, RuneError can slip in.  Actual RuneError
		// is okay, because the user who provided the text may only have the corrupted text to begin
		// with, and may provide U+FFFD.  To distinguish between a real U+FFFD, and a corrupted created
		// by Go, check the length of it.  In case it's bad, return an Internal rather than
		/// InvalidArgument.
		if r == utf8.RuneError {
			if _, ln := utf8.DecodeRuneInString(s[i:]); ln <= 1 {
				return status.Internalf(nil, "bad utf8 encoding at pos %d", i)
			}
		}
		if unicode.Is(notAllowedRange, r) {
			return status.InvalidArgumentf(nil, "unsupported code point %U at pos %d", r, i)
		}
	}
	return nil
}
