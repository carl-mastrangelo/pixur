package text

import (
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
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
func ValidateEncoding(text, fieldname string) error {
	return validateEncoding(text, fieldname)
}

func validateEncoding(text, fieldname string) status.S {
	if !utf8.ValidString(text) {
		return status.InvalidArgumentf(nil, "invalid %s utf8 text: '%s'", fieldname, text)
	}
	return nil
}

// ValidateCodepoints ensures that the string only contains reasonable characters.  By default,
// Unicode classes Cc (controls), Co (private use), Cs (surrogates), and Noncharacters are
// forbidden.  (\t, \r, and \n are allowed, though).   Allowed classes include L (letters), M
// (marks), N (numbers), P (punctuation), S (symbols), Z (separators/spaces), and Cf (formats).
// Additionally, reserved characters, i.e. those for future use are allowed, as they may become
// mapped in the future.
func ValidateCodepoints(text, fieldname string) error {
	return validateCodepoints(text, fieldname)
}

// validateCodepoints does a blacklist check rather than a whitelist.  Since the allowed and not
// allowed sets are finite and exclusive, it is equivalent to a whitelist check.  It's possible
// that the unicode spec could move some characters from the allowed range into the not allowed
// range (i.e. from reserved to control), but that's less bad than rejecting future valid sequences
// (i.e. reserved to letter).
func validateCodepoints(text, fieldname string) status.S {
	for i, r := range text {
		// Incase s hasn't been checked with validateEncoding, RuneError can slip in.  Actual RuneError
		// is okay, because the user who provided the text may only have the corrupted text to begin
		// with, and may provide U+FFFD.  To distinguish between a real U+FFFD, and a corrupted created
		// by Go, check the length of it.  In case it's bad, return an Internal rather than
		/// InvalidArgument.
		if r == utf8.RuneError {
			if _, ln := utf8.DecodeRuneInString(text[i:]); ln <= 1 {
				return status.Internalf(nil, "invalid %s utf8 text at pos %d", fieldname, i)
			}
		}
		if sts := validateCodepointUnsafe(i, r, fieldname); sts != nil {
			return sts
		}
	}
	return nil
}

func validateCodepointUnsafe(i int, r rune, fieldname string) status.S {
	if unicode.Is(notAllowedRange, r) {
		return status.InvalidArgumentf(nil, "unsupported char %U in %s at pos %d", r, fieldname, i)
	}
	return nil
}

// ValidateMaxBytes ensures that the string is bounded by min and max, inclusive.
func ValidateMaxBytes(text, fieldname string, min, max int64) error {
	return validateMaxBytes(text, fieldname, min, max)
}

func validateMaxBytes(text, fieldname string, min, max int64) status.S {
	if ln := int64(len(text)); ln < min {
		return status.InvalidArgumentf(nil, "%s too short (%d < %d)", fieldname, ln, min)
	} else if ln > max {
		return status.InvalidArgumentf(nil, "%s too long (%d > %d)", fieldname, ln, max)
	}
	return nil
}

// ToCanonical normalizes text to NFC, suitable for storage and transmission.
func ToCanonical(text, fieldname string) (string, error) {
	return toCanonical(text, fieldname)
}

func toCanonical(text, fieldname string) (string, status.S) {
	if sts := validateEncoding(text, fieldname); sts != nil {
		return "", sts
	}
	if sts := validateCodepoints(text, fieldname); sts != nil {
		return "", sts
	}
	return toCanonicalPrevalidated(text), nil
}

func toCanonicalPrevalidated(text string) string {
	return norm.NFC.String(text)
}
