package tasks

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/status"
)

// There is justification for several of the test symbols used here, but not in each function.

func TestValidateAndNormalizeText_normalizes(t *testing.T) {
	valid := func(_, _ string) status.S { return nil }
	s, sts := validateAndNormalizeText("A\u030A", "field", 0, 3, valid)
	if sts != nil {
		t.Fatal(sts)
	}
	if s != "\u00C5" {
		t.Fatal("did not normalize")
	}
}

func TestValidateAndNormalizeText_validatorFails(t *testing.T) {
	invalid := func(_, _ string) status.S { return status.InvalidArgument(nil, "expected") }
	_, sts := validateAndNormalizeText("a", "field", 0, 1, invalid)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "expected"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndNormalizeText_tooShortFails(t *testing.T) {
	valid := func(_, _ string) status.S { return nil }
	_, sts := validateAndNormalizeText("\U0001D160", "field", 5, 11, valid)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "too short"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndNormalizeText_invalidUtf8Fails(t *testing.T) {
	valid := func(_, _ string) status.S { return nil }
	_, sts := validateAndNormalizeText(string([]byte{0xCC, 0x00}), "field", 2, 11, valid)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "utf8 text"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndNormalizeText_tooLongAfterNormFails(t *testing.T) {
	valid := func(_, _ string) status.S { return nil }
	_, sts := validateAndNormalizeText("\U0001D160", "field", 4, 11, valid)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "too long"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndNormalizeText_tooShortAfterNormFails(t *testing.T) {
	valid := func(_, _ string) status.S { return nil }
	_, sts := validateAndNormalizeText("A\u030A", "field", 3, 12, valid)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "too short"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateMaxLength(t *testing.T) {
	sts := validateMaxLength("a", "field", 1, 1)
	if sts != nil {
		t.Error(sts)
	}
}

func TestValidateMaxLength_tooShort(t *testing.T) {
	sts := validateMaxLength("a", "field", 2, 2)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "too short"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateMaxLength_tooLong(t *testing.T) {
	sts := validateMaxLength("a", "field", 0, 0)
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "too long"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestNormalizeUnicodeText_failsInvalidUtf8(t *testing.T) {
	_, sts := normalizeUnicodeText(string([]byte{0xCC, 0x00}), "field")
	if sts == nil {
		t.Fatal(sts)
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "utf8 text"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestNormalizeUnicodeText_expanding(t *testing.T) {
	// From http://unicode.org/faq/normalization.html
	// Some text can expand when being normalized.  This case goes from
	// 4 bytes to 12 bytes in UTF-8.
	s, sts := normalizeUnicodeText("\U0001D160", "field") // 𝅘𝅥𝅮
	if sts != nil {
		t.Fatal(sts)
	}
	if len(s) != 12 {
		t.Fatal("did not normalize", []byte(s))
	}
}

func TestNormalizeUnicodeText(t *testing.T) {
	// From http://unicode.org/reports/tr15/
	s, sts := normalizeUnicodeText("A\u030A", "field")
	if sts != nil {
		t.Fatal(sts)
	}
	if s != "\u00C5" {
		t.Fatal("did not normalize")
	}
}

func TestValidateGraphicText_nonGraphicFails(t *testing.T) {
	sts := validateGraphicText("\u0000", "field")
	if sts == nil {
		t.Fatal("expected an error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "nongraphic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateGraphicText_singleCodepointEmojiWorks(t *testing.T) {
	sts := validateGraphicText("😺😇😊😳😈☠👁", "field")
	if sts != nil {
		t.Fatal(sts)
	}
}

// TODO: this should succeed, but the code isn't smart enough to handle it yet.
func TestValidateGraphicText_multiCodepointEmojiFails(t *testing.T) {
	// Man shrugging, per https://unicode.org/emoji/charts/full-emoji-list.html
	sts := validateGraphicText("\U0001F937\u200D\u2642\uFE0F", "field")
	if sts == nil {
		t.Fatal("expected an error")
	}
}

func TestValidateGraphicText_commonWhitespaceWorks(t *testing.T) {
	sts := validateGraphicText("\r\n\t ", "field")
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestValidateGraphicText_plainTextWorks(t *testing.T) {
	sts := validateGraphicText("The quick brown fox jumps over the lazy dog", "field")
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestValidatePrintText_nonPrintFails(t *testing.T) {
	sts := validatePrintText("\u0000", "field")
	if sts == nil {
		t.Fatal("expected an error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unprintable"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidatePrintText_singleCodepointEmojiWorks(t *testing.T) {
	sts := validatePrintText("😺😇😊😳😈☠👁", "field")
	if sts != nil {
		t.Fatal(sts)
	}
}

// TODO: this should succeed, but the code isn't smart enough to handle it yet.
func TestValidatePrintText_multiCodepointEmojiFails(t *testing.T) {
	// Man shrugging, per https://unicode.org/emoji/charts/full-emoji-list.html
	sts := validatePrintText("\U0001F937\u200D\u2642\uFE0F", "field")
	if sts == nil {
		t.Fatal("expected an error")
	}
}

func TestValidatePrintText_commonWhitespaceFails(t *testing.T) {
	sts := validatePrintText("\r\n\t ", "field")
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unsupported whitespace"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidatePrintText_plainTextWorks(t *testing.T) {
	sts := validatePrintText("The quick brown fox jumps over the lazy dog", "field")
	if sts != nil {
		t.Fatal(sts)
	}
}
