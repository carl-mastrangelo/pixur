// Package text implements text processing and validation.
package text // import "pixur.org/pixur/be/text"

import (
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/unicode/rangetable"

	"pixur.org/pixur/be/status"
)

const UnicodeVersion = unicode.Version

// TextValidator validates some text
type TextValidator func(text, fieldname string) error

// TextNormalizer normalizes some text
type TextNormalizer func(text, fieldname string) (string, error)

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

var noNewlineRange *unicode.RangeTable

func init() {
	_, ncc := buildCc()
	notAllowedRange = rangetable.Merge(ncc, unicode.Co, unicode.Cs, unicode.Noncharacter_Code_Point)

	noNewline := &unicode.RangeTable{
		R16: []unicode.Range16{
			{Lo: 0x0a, Hi: 0x0a, Stride: 1}, // \r
			{Lo: 0x0d, Hi: 0x0d, Stride: 1}, // \n
		},
		LatinOffset: 2,
	}
	noNewlineRange = rangetable.Merge(noNewline, unicode.Zl, unicode.Zp)
}

var _ TextValidator = ValidateEncoding

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

var _ TextValidator = ValidateCodepoints

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

// MaxBytesValidator produces a TextValidator that checks text length in bytes.
func MaxBytesValidator(min, max int64) TextValidator {
	return func(text, fieldname string) error {
		return ValidateMaxBytes(text, fieldname, min, max)
	}
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

var _ TextNormalizer = ToNFC

// ToNFC normalizes text to NFC, suitable for storage and transmission.
func ToNFC(text, fieldname string) (string, error) {
	return toNFC(text, fieldname)
}

func toNFC(text, fieldname string) (string, status.S) {
	if sts := validateEncoding(text, fieldname); sts != nil {
		return "", sts
	}
	if sts := validateCodepoints(text, fieldname); sts != nil {
		return "", sts
	}
	return toNFCUnsafe(text), nil
}

var _ TextNormalizer = ToNFCUnsafe

// ToNFCUnsafe converts prevalidated text into NFC.  Text *must* have been previously
// validated in order to use this function.  This never returns a non-nil error, and ignores the
// fieldname.
func ToNFCUnsafe(text, fieldname string) (string, error) {
	return toNFCUnsafe(text), nil
}

func toNFCUnsafe(text string) string {
	return norm.NFC.String(text)
}

// Validate validates text.
func Validate(text, fieldname string, validators ...TextValidator) error {
	return validate(text, fieldname, validators)
}

func validate(text, fieldname string, validators []TextValidator) status.S {
	for _, v := range validators {
		if err := v(text, fieldname); err != nil {
			return status.From(err)
		}
	}
	return nil
}

// Normalize normalizes text.
func Normalize(text, fieldname string, normalizers ...TextNormalizer) (string, error) {
	return normalize(text, fieldname, normalizers)
}

func normalize(text, fieldname string, normalizers []TextNormalizer) (string, status.S) {
	for _, n := range normalizers {
		var err error
		if text, err = n(text, fieldname); err != nil {
			return "", status.From(err)
		}
	}
	return text, nil
}

// ValidateAndNormalize validates text, normalizes it, and then revalidates it.
func ValidateAndNormalize(
	text, fieldname string, textnorm TextNormalizer, validators ...TextValidator) (string, error) {
	return validateAndNormalize(text, fieldname, []TextNormalizer{textnorm}, validators, validators)
}

// ValidateAndNormalize validates text, normalizes it, and then revalidates it.
func ValidateAndNormalizeMulti(
	text, fieldname string, textnorms []TextNormalizer, validators ...TextValidator) (string, error) {
	return validateAndNormalize(text, fieldname, textnorms, validators, validators)
}

func validateAndNormalize(text, fieldname string, ns []TextNormalizer, prevalid []TextValidator,
	postvalid []TextValidator) (string, status.S) {
	if sts := validate(text, fieldname, prevalid); sts != nil {
		return "", sts
	}
	newtext, sts := normalize(text, fieldname, ns)
	if sts != nil {
		return "", sts
	}
	if sts := validate(newtext, fieldname, postvalid); sts != nil {
		return "", sts
	}
	return newtext, nil
}

// DefaultValidateAndNormalize performs regular normalization.  Use this by default.
func DefaultValidateAndNormalize(
	text, fieldname string, minbytes, maxbytes int64, extra ...TextValidator) (string, error) {
	return defaultValidateAndNormalize(text, fieldname, minbytes, maxbytes, nil, extra)
}

func defaultValidateAndNormalize(text, fieldname string, minbytes, maxbytes int64,
	unsafePre, extra []TextValidator) (string, status.S) {
	pre, post := defaultValidators(minbytes, maxbytes)
	pre, post = append(append(pre, unsafePre...), extra...), append(post, extra...)
	return validateAndNormalize(text, fieldname, []TextNormalizer{ToNFCUnsafe}, pre, post)
}

// DefaultValidator produces a TextValidator that checks for the default validation
func DefaultValidator(min, max int64) TextValidator {
	return func(text, fieldname string) error {
		return DefaultValidate(text, fieldname, min, max)
	}
}

// DefaultValidate performs regular validation.  Use this by default
func DefaultValidate(text, fieldname string, minbytes, maxbytes int64) error {
	return defaultValidate(text, fieldname, minbytes, maxbytes)
}

func defaultValidate(text, fieldname string, minbytes, maxbytes int64) status.S {
	prevalid, _ := defaultValidators(minbytes, maxbytes)
	return validate(text, fieldname, prevalid)
}

// defaultValidators builds all the default validatos.  post is always a subsequence pre.
func defaultValidators(min, max int64, unsafePre ...TextValidator) (pre, post []TextValidator) {
	post = append(post, MaxBytesValidator(min, max))
	pre = append(pre, post...)
	pre = append(pre, ValidateEncoding, ValidateCodepoints)
	pre = append(pre, unsafePre...)
	return
}

// ValidateNoNewlines ensures there are no newlines or other vertical spacing characters in
// the string.  This includes \r, \n, and Unicode classes Zl and Zp.
func ValidateNoNewlines(text, fieldname string) error {
	return validateNoNewlines(text, fieldname)
}

func validateNoNewlines(text, fieldname string) status.S {
	if sts := validateEncoding(text, fieldname); sts != nil {
		return sts
	}
	if sts := validateCodepoints(text, fieldname); sts != nil {
		return sts
	}
	return validateNoNewlinesUnsafe(text, fieldname)
}

// ValidateNoNewlinesUnsafe ensures there are no newlines or other vertical spacing characters in
// the string.  This includes \r, \n, and Unicode classes Zl, and Zp.  Text *must* have been
// previously validated in order to use this function.
func ValidateNoNewlinesUnsafe(text, fieldname string) error {
	return validateNoNewlinesUnsafe(text, fieldname)
}

func validateNoNewlinesUnsafe(text, fieldname string) status.S {
	for i, r := range text {
		if unicode.Is(noNewlineRange, r) {
			return status.InvalidArgumentf(nil, "unsupported newline %U in %s at pos %d", r, fieldname, i)
		}
	}
	return nil
}

// DefaultValidateNoNewlineAndNormalize performs regular normalization and fails on Newlines.
func DefaultValidateNoNewlineAndNormalize(
	text, fieldname string, minbytes, maxbytes int64, extra ...TextValidator) (string, error) {
	unsafePre := []TextValidator{ValidateNoNewlinesUnsafe}
	return defaultValidateAndNormalize(text, fieldname, minbytes, maxbytes, unsafePre, extra)
}

var _ TextNormalizer = ToCaselessNFKC

// ToCaselessCompatible normalizes to NFKC and casefolds text for comparison.
func ToCaselessNFKC(text, fieldname string) (string, error) {
	return toCaselessNFKC(text, fieldname)
}

func toCaselessNFKC(text, fieldname string) (string, status.S) {
	if sts := validateEncoding(text, fieldname); sts != nil {
		return "", sts
	}
	if sts := validateCodepoints(text, fieldname); sts != nil {
		return "", sts
	}
	return toCaselessNFKCUnsafe(text), nil
}

var _ TextNormalizer = ToCaselessNFKCUnsafe

// ToCaselessNFKCUnsafe converts prevalidated text into caseless NFKC.  Text *must* have been
// previously validated in order to use this function.  This never returns a non-nil error, and
// ignores the fieldname.
func ToCaselessNFKCUnsafe(text, fieldname string) (string, error) {
	return toCaselessNFKCUnsafe(text), nil
}

// toCaselessNFKCUnsafe implements caseless compatibility matching in Unicode 11.0, Ch 3, D146.
func toCaselessNFKCUnsafe(text string) string {
	fold, data := cases.Fold(), []byte(text)
	// TR 15 Section 7 says NFKC(NFKD(x)) == NFKC(x), so the outer most NFKD becomes an NFKC
	return string(norm.NFKC.Bytes(fold.Bytes(norm.NFKD.Bytes(fold.Bytes(norm.NFD.Bytes(data))))))
}
