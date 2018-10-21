package tasks

import (
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"pixur.org/pixur/be/status"
)

// TODO: test

type textValidator func(text, fieldname string) status.S

// for text that cannot contain newlines
func validateAndNormalizePrintText(text, fieldname string, min, max int) (string, status.S) {
	return validateAndNormalizeText(text, fieldname, min, max, validatePrintText)
}

// for text that can contain newlines
func validateAndNormalizeGraphicText(text, fieldname string, min, max int) (string, status.S) {
	return validateAndNormalizeText(text, fieldname, min, max, validateGraphicText)
}

func validateAndNormalizeText(text, fieldname string, min, max int, validate textValidator) (
	string, status.S) {
	if sts := validateMaxLength(text, fieldname, min, max); sts != nil {
		return "", sts
	}
	if sts := validateUtf8(text, fieldname); sts != nil {
		return "", sts
	}
	newtext := normalizeUnicodeTextUnsafe(text)
	if sts := validateMaxLength(newtext, fieldname, min, max); sts != nil {
		return "", sts
	}
	if sts := validate(newtext, fieldname); sts != nil {
		return "", sts
	}
	return newtext, nil
}

func validateUtf8(text, fieldname string) status.S {
	if !utf8.ValidString(text) {
		return status.InvalidArgumentf(nil, "invalid %s utf8 text: '%s'", fieldname, text)
	}
	return nil
}

func validateMaxLength(text, fieldname string, min, max int) status.S {
	if len(text) < min {
		return status.InvalidArgumentf(nil, "%s too short (%d < %d)", fieldname, len(text), min)
	} else if len(text) > max {
		return status.InvalidArgumentf(nil, "%s too long (%d > %d)", fieldname, len(text), max)
	}
	return nil
}

func normalizeUnicodeText(text, fieldname string) (string, status.S) {
	if !utf8.ValidString(text) {
		return "", status.InvalidArgumentf(nil, "invalid %s utf8 text: '%s'", fieldname, text)
	}
	return normalizeUnicodeTextUnsafe(text), nil
}

func normalizeUnicodeTextUnsafe(text string) string {
	return norm.NFC.String(text)
}

// for text that cannot contain newlines
func validatePrintText(text, fieldname string) status.S {
	for i, r := range text {
		if !unicode.IsPrint(r) {
			var msg string
			if unicode.IsSpace(r) {
				msg = "unsupported whitespace"
			} else {
				msg = "unprintable"
			}
			return status.InvalidArgumentf(nil, "%s rune '%U' in %s @%d", msg, r, fieldname, i)
		}
	}
	return nil
}

// for text that can contain newlines
func validateGraphicText(text, fieldname string) status.S {
	for i, r := range text {
		if !unicode.IsGraphic(r) {
			msg := "nongraphic"
			return status.InvalidArgumentf(nil, "%s rune '%U' in %s @%d", msg, r, fieldname, i)
		}
	}
	return nil
}
