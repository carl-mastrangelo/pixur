// The idea behind this is to use variable length coding to get some nice numeric
// properties.  To encode, slice up an integer into 4 bit chunks with the most significant
// group (MSG)first  (chunks are aligned on 4 bit boundaries).  Then, for all but the last group,
// prepend a 1 bit;  the last group gets a 0 instead.  This makes each group a 5 bit value
// that corresponds to one of 32 ASCII characters.

// For example, to encode the number 300,  break it up into 0001 0010 1100.  (256 + 2*16 + 12)
// Next add the continuation bit: 10001 10010 01100.  Last, look up the ASCII character at each
// group value 17 => h, 18 => j, 12 => c.  This gives the encoding "hjc".

// There are a few nice properties of this encoding:
// - Variable length, extensible, can safely encode any length number (even 128 bit ints)
// - Unambiguous, can concatenate numbers together and still be able to parse them
// - The 32 ASCII values are chosen to avoid similar looking letters, it is case insensitive
// - Avoids vowels i, o, and u, avoiding most curse words.
// - Slightly more dense than base 10,
// - The MSG ordering causes larger numbers to be sortable by prefix.

// There are also some downsides.  These aren't major though, and can be worked around.
// - Negative numbers encode to very large values.  (can be zig zag encoded if need be)
// - The high order characters G - Z are used far more often than low order ones.

// Inspiration for this coding comes from Protocol Buffers, which use 8 bit LSG encoding.  The
// alphabet is from Douglas Crockford's Base 32 encoding, but without dash or error checking
// parts.

package schema

import (
	"encoding"
	"fmt"

	stdb32 "encoding/base32"
)

var (
	b32DecodeTable []byte = b32DecodeValues()
	b32EncodeTable []byte = b32EncodeValues()

	_ encoding.TextMarshaler   = new(B32Varint)
	_ encoding.TextUnmarshaler = new(B32Varint)
)

type B32Varint int64

func (v *B32Varint) MarshalText() ([]byte, error) {
	// This can handle negatives, but they will be huge.
	num := uint64(*v)
	// The 4 bit groups, least significant group first.  Make room for a 16  4-bit groups.
	lsgBuf := make([]byte, 0, 16)

	lsgBuf = append(lsgBuf, b32EncodeTable[num&0xF])
	num >>= 4

	for ; num > 0; num >>= 4 {
		lsgBuf = append(lsgBuf, b32EncodeTable[num&0xF|0x10])
	}

	// All the characters are in place, time to reverse the order
	for i := 0; i < len(lsgBuf)/2; i++ {
		lsgBuf[i], lsgBuf[len(lsgBuf)-i-1] = lsgBuf[len(lsgBuf)-i-1], lsgBuf[i]
	}

	return lsgBuf, nil
}

func (v *B32Varint) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return fmt.Errorf("Empty B32Varint input")
	}
	// Too big of a number
	if len(text) > 16 {
		return stdb32.CorruptInputError(17)
	}
	// Overlong encoding, leading zeros
	if b32DecodeTable[text[0]] == 0x10 {
		return stdb32.CorruptInputError(0)
	}

	var num uint64
	for i, b := range text {
		val := b32DecodeTable[b]
		if val == 0xFF {
			return stdb32.CorruptInputError(i)
		}

		num = num<<4 | uint64(val&0xF)

		if val&0x10 == 0 {
			// too much data
			if len(text) > i+1 {
				return stdb32.CorruptInputError(i + 1)
			}
			break
		} else {
			// They didnt have a finishing byte
			if i == len(text)-1 {
				return stdb32.CorruptInputError(i)
			}
		}
	}

	*v = B32Varint(num)
	return nil
}

func b32DecodeValues() []byte {
	vals := make([]byte, 256)
	for i := 0; i < len(vals); i++ {
		vals[i] = 0xFF
	}
	lookup := map[rune]byte{
		'0': 0, 'O': 0, 'o': 0,
		'1': 1, 'I': 1, 'i': 1, 'L': 1, 'l': 1,
		'2': 2,
		'3': 3,
		'4': 4,
		'5': 5,
		'6': 6,
		'7': 7,
		'8': 8,
		'9': 9,
		'A': 10, 'a': 10,
		'B': 11, 'b': 11,
		'C': 12, 'c': 12,
		'D': 13, 'd': 13,
		'E': 14, 'e': 14,
		'F': 15, 'f': 15,
		'G': 16, 'g': 16,
		'H': 17, 'h': 17,
		'J': 18, 'j': 18,
		'K': 19, 'k': 19,
		'M': 20, 'm': 20,
		'N': 21, 'n': 21,
		'P': 22, 'p': 22,
		'Q': 23, 'q': 23,
		'R': 24, 'r': 24,
		'S': 25, 's': 25,
		'T': 26, 't': 26,
		'V': 27, 'v': 27,
		'W': 28, 'w': 28,
		'X': 29, 'x': 29,
		'Y': 30, 'y': 30,
		'Z': 31, 'z': 31,
	}

	for c, v := range lookup {
		vals[int(c)] = v
	}

	return vals
}

func b32EncodeValues() []byte {
	charTable := []rune{
		'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
		'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'j', 'k',
		'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x',
		'y', 'z',
	}

	b := make([]byte, len(charTable))
	for i, c := range charTable {
		b[i] = byte(c)
	}
	return b
}
