package text

import (
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/unicode/rangetable"
	"google.golang.org/grpc/codes"
)

// TODO: check this for casefold equiv
//0041+0301+0328 = 0041+0328+0301

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
		string([]byte{0xC0, 0x00}),
		string([]byte{0xC2}),
		string([]byte{0xF7, 0xBF, 0xBF, 0xBF}), // over max codepoint
		string([]byte{0xED, 0xa0, 0x80}),       // surrogate
	}
	for _, s := range valid {
		sts := validateEncoding(s)
		if sts != nil {
			t.Error("failed on", s, sts)
		}
	}
	for _, s := range invalid {
		sts := validateEncoding(s)
		if sts == nil {
			t.Fatal("expected error", s)
		}
		if have, want := sts.Code(), codes.InvalidArgument; have != want {
			t.Error("have", have, "want", want, "for", s)
		}
		if have, want := sts.Message(), "bad utf8"; !strings.Contains(have, want) {
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
		sts := validateCodepoints(s)
		if sts != nil {
			t.Error("failed on", s, sts)
		}
	}
	for i, s := range invalid {
		sts := validateCodepoints(s)
		if sts == nil {
			t.Fatal("expected error", i, s)
		}
		if have, want := sts.Code(), codes.InvalidArgument; have != want {
			t.Error("have", have, "want", want, "for", i, s)
		}
		if have, want := sts.Message(), "unsupported code point"; !strings.Contains(have, want) {
			t.Error("have", have, "want", want, "for", i, s)
		}
	}
	for i, s := range badutf8 {
		sts := validateCodepoints(s)
		if sts == nil {
			t.Fatal("expected error", i, s)
		}
		if have, want := sts.Code(), codes.Internal; have != want {
			t.Error("have", have, "want", want, "for", i, s)
		}
		if have, want := sts.Message(), "bad utf8 encoding"; !strings.Contains(have, want) {
			t.Error("have", have, "want", want, "for", i, s)
		}
	}
}
