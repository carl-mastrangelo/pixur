package text

import (
	"testing"
	"unicode"

	"golang.org/x/text/unicode/rangetable"
)

// TODO: check this for casefold equiv
//0041+0301+0328 = 0041+0328+0301

func TestAllowedChars(t *testing.T) {
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
