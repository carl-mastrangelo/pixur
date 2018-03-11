package schema

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	// bits per symbol
	bits = 5
	// number of possible symbols
	symbolCount = 1 << bits
)

var (
	// Fancy definition of weyyyyyyyyyyyf.
	maxValue  = Varint(-1).EncodeBytes()
	maxLength = len(maxValue)
)

// Prefix counts.
const (
	prefixg = (1<<(bits*iota)-1)*symbolCount/(symbolCount-1) + (symbolCount / 2)
	prefixh
	prefixj
	prefixk
	prefixm
	prefixn
	prefixp
	prefixq
	prefixr
	prefixs
	prefixt
	prefixv
	prefixw
	// These are defined, but not used, since they are bigger than uint64 max.
	prefixx
	prefixy
	prefixz
)

var prefixDecodeTable = [256]uint64{
	'0': 0,
	'1': 1,
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

	'G': prefixg, 'g': prefixg,
	'H': prefixh, 'h': prefixh,
	'J': prefixj, 'j': prefixj,
	'K': prefixk, 'k': prefixk,
	'M': prefixm, 'm': prefixm,
	'N': prefixn, 'n': prefixn,
	'P': prefixp, 'p': prefixp,
	'Q': prefixq, 'q': prefixq,
	'R': prefixr, 'r': prefixr,
	'S': prefixs, 's': prefixs,
	'T': prefixt, 't': prefixt,
	'V': prefixv, 'v': prefixv,
	'W': prefixw, 'w': prefixw,
}

var prefixEncodeTable = [32]uint64{
	16: prefixg,
	17: prefixh,
	18: prefixj,
	19: prefixk,
	20: prefixm,
	21: prefixn,
	22: prefixp,
	23: prefixq,
	24: prefixr,
	25: prefixs,
	26: prefixt,
	27: prefixv,
	28: prefixw,
}

// Number of symbols to expect, given the prefix.
var lengthTable = [256]int{
	'0': 1,
	'1': 1,
	'2': 1,
	'3': 1,
	'4': 1,
	'5': 1,
	'6': 1,
	'7': 1,
	'8': 1,
	'9': 1,

	'A': 1, 'a': 1,
	'B': 1, 'b': 1,
	'C': 1, 'c': 1,
	'D': 1, 'd': 1,
	'E': 1, 'e': 1,
	'F': 1, 'f': 1,

	'G': 2, 'g': 2,
	'H': 3, 'h': 3,
	'J': 4, 'j': 4,
	'K': 5, 'k': 5,
	'M': 6, 'm': 6,
	'N': 7, 'n': 7,
	'P': 8, 'p': 8,
	'Q': 9, 'q': 9,
	'R': 10, 'r': 10,
	'S': 11, 's': 11,
	'T': 12, 't': 12,
	'V': 13, 'v': 13,
	'W': 14, 'w': 14,
	// These are defined in order to give a better error message.
	'X': 15, 'x': 15,
	'Y': 16, 'y': 16,
	'Z': 17, 'z': 17,
}

var valueTable = [256]uint64{
	'0': 0,
	'1': 1,
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

var encodeTable = [32]byte{
	0:  '0',
	1:  '1',
	2:  '2',
	3:  '3',
	4:  '4',
	5:  '5',
	6:  '6',
	7:  '7',
	8:  '8',
	9:  '9',
	10: 'a',
	11: 'b',
	12: 'c',
	13: 'd',
	14: 'e',
	15: 'f',
	16: 'g',
	17: 'h',
	18: 'j',
	19: 'k',
	20: 'm',
	21: 'n',
	22: 'p',
	23: 'q',
	24: 'r',
	25: 's',
	26: 't',
	27: 'v',
	28: 'w',
	29: 'x',
	30: 'y',
	31: 'z',
}

type Varint int64

func (v Varint) Append(dest []byte) []byte {
	n := uint64(v)
	if n < 0x10 {
		return append(dest, encodeTable[n])
	}
	var suffix []byte
	if n < prefixw {
		for i := 0x10; i < len(prefixEncodeTable); i++ {
			if n < prefixEncodeTable[i+1] {
				n -= prefixEncodeTable[i]
				suffix = make([]byte, lengthTable[encodeTable[i]])
				suffix[0] = encodeTable[i]
				break
			}
		}
	} else {
		n -= prefixw
		suffix = make([]byte, lengthTable['w'])
		suffix[0] = 'w'
	}
	for i := len(suffix) - 1; i >= 1; i-- {
		suffix[i] = encodeTable[n&(symbolCount-1)]
		n >>= bits
	}

	return append(dest, suffix...)
}

func (v Varint) EncodeBytes() []byte {
	return v.Append(nil)
}

func (v Varint) Encode() string {
	return string(v.EncodeBytes())
}

var (
	errNoInput       = errors.New("varint: no input")
	errInvalidLength = errors.New("varint: invalid length")
	errInvalidSymbol = errors.New("varint: invalid symbol")
	errEof           = errors.New("varint: eof")
	errOverflow      = errors.New("varint: overflow")
	errExcessInput   = errors.New("varint: excess input")
)

// DecodeBytes sets v to the value of raw, and returns the number of bytes consumed.
func (v *Varint) DecodeBytes(raw []byte) (int, error) {
	if len(raw) == 0 {
		return 0, errNoInput
	}
	length := lengthTable[raw[0]]
	if length == 0 {
		return 0, errInvalidLength
	}
	if length > len(raw) {
		return 0, errEof
	}
	if length >= maxLength && bytes.Compare(bytes.ToLower(raw), maxValue) > 0 {
		return 0, errOverflow
	}

	var num uint64
	for i := 1; i < length; i++ {
		val := valueTable[raw[i]]
		if val == 0 && raw[i] != '0' {
			return 0, errInvalidSymbol
		}
		num = num<<5 + val
	}

	*v = Varint(int64(num + prefixDecodeTable[raw[0]]))
	return length, nil
}

func (v *Varint) Decode(raw string) (int, error) {
	return v.DecodeBytes([]byte(raw))
}

func (v *Varint) DecodeAll(raw string) error {
	var tmp Varint
	n, err := tmp.Decode(raw)
	if err != nil {
		return err
	}
	if n != len(raw) {
		return errExcessInput
	}
	// Don't overwrite it on error
	*v = tmp
	return nil
}

func (v Varint) String() string {
	return fmt.Sprintf("%s(%d)", v.Encode(), v)
}
