package text

import (
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/unicode/rangetable"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/status"
)

// There is justification for several of the test symbols used here, but not in each function.

func TestAllowedCodePointsExclusive(t *testing.T) {
	allowedcc, _ := buildCc()
	allowed := rangetable.Merge(
		unicode.Letter,
		unicode.Mark,
		unicode.Number,
		unicode.Punct,
		unicode.Symbol,
		unicode.Zs, // Spaces
		// Control / Formatting chars:
		unicode.Zl,
		unicode.Zp,
		unicode.Cf,
		allowedcc)

	reserved := 0
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if a, n := unicode.Is(allowed, r), unicode.Is(notAllowedRange, r); n && a {
			t.Fatalf("bad %U", r)
		} else if !n && !a {
			reserved++
		}
	}
	// Unicode 10.0 says it's this much
	if have, want := reserved, 837775; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestValidateEncoding(t *testing.T) {
	valid := []string{
		"\u0000",                               // null
		"foobar",                               // regular
		"\uFFFD",                               // replacement char
		string([]byte{0xF3, 0xBF, 0xBF, 0xBF}), // max codepoint
		"\uF8FF",                               // private use
	}
	invalid := []string{
		string([]byte{0xCC, 0x00}),
		string([]byte{0xC2}),
		string([]byte{0xF7, 0xBF, 0xBF, 0xBF}), // over max codepoint
		string([]byte{0xED, 0xa0, 0x80}),       // surrogate
	}
	for _, s := range valid {
		sts := validateEncoding(s, "field")
		if sts != nil {
			t.Error("failed on", s, sts)
		}
	}
	for _, s := range invalid {
		sts := validateEncoding(s, "field")
		if sts == nil {
			t.Fatal("expected error", s)
		}
		if have, want := sts.Code(), codes.InvalidArgument; have != want {
			t.Error("have", have, "want", want, "for", s)
		}
		if have, want := sts.Message(), "utf8 text"; !strings.Contains(have, want) {
			t.Error("have", have, "want", want, "for", s)
		}
	}
}

func TestValidateCodepoints(t *testing.T) {
	valid := []string{
		"\t\r\n",                                 // special control
		"foobar\U000E0100",                       // mark
		"  123!@#.  ",                            // number, symbol, punct, space
		"\U0001F441\uFE0F\u200D\U0001F5E8\uFE0F", // emoji, format
		"\uFFFD\uFFFC",                           // replacement chars
		"\U000A0000",                             // Reserved (as of Unicode 11)
	}
	invalid := []string{
		"\u0000",                               // null (control)
		"\uF8FF",                               // private use
		string([]byte{0xF3, 0xBF, 0xBF, 0xBF}), // max codepoint (non character)
	}
	badutf8 := []string{
		string([]byte{0xED, 0xa0, 0x80}),       // surrogate
		string([]byte{0xF7, 0xBF, 0xBF, 0xBF}), // over max codepoint
	}
	for _, s := range valid {
		sts := validateCodepoints(s, "field")
		if sts != nil {
			t.Error("failed on", s, sts)
		}
	}
	for i, s := range invalid {
		sts := validateCodepoints(s, "field")
		if sts == nil {
			t.Fatal("expected error", i, s)
		}
		if have, want := sts.Code(), codes.InvalidArgument; have != want {
			t.Error("have", have, "want", want, "for", i, s)
		}
		if have, want := sts.Message(), "unsupported char"; !strings.Contains(have, want) {
			t.Error("have", have, "want", want, "for", i, s)
		}
	}
	for i, s := range badutf8 {
		sts := validateCodepoints(s, "field")
		if sts == nil {
			t.Fatal("expected error", i, s)
		}
		if have, want := sts.Code(), codes.Internal; have != want {
			t.Error("have", have, "want", want, "for", i, s)
		}
		if have, want := sts.Message(), "utf8 text"; !strings.Contains(have, want) {
			t.Error("have", have, "want", want, "for", i, s)
		}
	}
}

func TestValidateMaxBytes(t *testing.T) {
	sts := validateMaxBytes("a", "field", 1, 1)
	if sts != nil {
		t.Error(sts)
	}
}

func TestValidateMaxBytes_tooShort(t *testing.T) {
	sts := validateMaxBytes("a", "field", 2, 2)
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

func TestValidateMaxBytes_tooLong(t *testing.T) {
	sts := validateMaxBytes("a", "field", 0, 0)
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

func TestToNFC_failsInvalidUtf8(t *testing.T) {
	_, sts := toNFC(string([]byte{0xCC, 0x00}), "field")
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

func TestToNFC_expanding(t *testing.T) {
	// From http://unicode.org/faq/normalization.html
	// Some text can expand when being normalized.  This case goes from
	// 4 bytes to 12 bytes in UTF-8.
	s, sts := toNFC("\U0001D160", "field") // 𝅘𝅥𝅮
	if sts != nil {
		t.Fatal(sts)
	}
	if len(s) != 12 {
		t.Fatal("did not normalize", []byte(s))
	}
}

func TestToNFC(t *testing.T) {
	// From http://unicode.org/reports/tr15/
	s, sts := toNFC("A\u030A", "field")
	if sts != nil {
		t.Fatal(sts)
	}
	if s != "\u00C5" {
		t.Fatal("did not normalize")
	}
}

func TestToNFC_norms(t *testing.T) {
	s1, sts := toNFC("\u0041\u0301\u0328", "field")
	if sts != nil {
		t.Fatal(sts)
	}
	s2, sts := toNFC("\u0041\u0328\u0301", "field")
	if sts != nil {
		t.Fatal(sts)
	}
	if s1 != s2 {
		t.Fatal("did not normalize", s1, s2)
	}
}

func TestDefaultValidateAndNormalize_normalizes(t *testing.T) {
	valid := func(_, _ string) error { return nil }
	s, sts := defaultValidateAndNormalize("A\u030A", "field", 0, 3, valid)
	if sts != nil {
		t.Fatal(sts)
	}
	if s != "\u00C5" {
		t.Fatal("did not normalize")
	}
}

func TestDefaultValidateAndNormalize_validatorFails(t *testing.T) {
	invalid := func(_, _ string) error { return status.InvalidArgument(nil, "expected") }
	_, sts := defaultValidateAndNormalize("a", "field", 0, 1, invalid)
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

func TestDefaultValidateAndNormalize_tooShortFails(t *testing.T) {
	valid := func(_, _ string) error { return nil }
	_, sts := defaultValidateAndNormalize("\U0001D160", "field", 5, 11, valid)
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

func TestDefaultValidateAndNormalize_invalidUtf8Fails(t *testing.T) {
	valid := func(_, _ string) error { return nil }
	_, sts := defaultValidateAndNormalize(string([]byte{0xCC, 0x00}), "field", 2, 11, valid)
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

func TestDefaultValidateAndNormalize_tooLongAfterNormFails(t *testing.T) {
	valid := func(_, _ string) error { return nil }
	_, sts := defaultValidateAndNormalize("\U0001D160", "field", 4, 11, valid)
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

func TestDefaultValidateAndNormalize_tooShortAfterNormFails(t *testing.T) {
	valid := func(_, _ string) error { return nil }
	_, sts := defaultValidateAndNormalize("A\u030A", "field", 3, 12, valid)
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

func TestDefaultValidateAndNormalize_nonGraphicFails(t *testing.T) {
	_, sts := defaultValidateAndNormalize("\u0000", "field", 0, 1)
	if sts == nil {
		t.Fatal("expected an error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unsupported char"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestDefaultValidateAndNormalize_singleCodepointEmojiWorks(t *testing.T) {
	_, sts := defaultValidateAndNormalize("😺😇😊😳😈☠👁", "field", 1, 100)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestDefaultValidateAndNormalize_multiCodepointEmojiFails(t *testing.T) {
	// Man shrugging, per https://unicode.org/emoji/charts/full-emoji-list.html
	_, sts := defaultValidateAndNormalize("\U0001F937\u200D\u2642\uFE0F", "field", 1, 100)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestDefaultValidateAndNormalize_commonWhitespaceWorks(t *testing.T) {
	_, sts := defaultValidateAndNormalize("\r\n\t ", "field", 1, 100)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestDefaultValidateAndNormalize_plainTextWorks(t *testing.T) {
	_, sts := defaultValidateAndNormalize("The quick brown fox jumps over the lazy dog", "field", 1, 100)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestDefaultValidateAndNormalize_customValidator(t *testing.T) {
	_, sts := defaultValidateAndNormalize("\r\n", "field", 1, 100, func(_, _ string) error {
		return status.InvalidArgument(nil, "unsupported whitespace")
	})
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "unsupported whitespace"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
