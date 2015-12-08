package schema

import (
	"bytes"
	"fmt"
)

type Varint int64

var (
	varintEncodeTable, varintDecodeTable = buildCodingTables()
)

func (v Varint) EncodeBytes() (raw []byte) {
	n := uint64(v)
	// Fast path, small numbers use the lower bytes
	if n < 0x10 {
		return []byte{varintEncodeTable[byte(n)]}
	}
	n -= 0x10
	// Length is tricky.  It represents the number of groups +1.  Thus, a length
	// of 0 means 1 group, 1 means 2 groups.  This is a side effect of cramming
	// the first 16 (0-15) digits in the low order lengths.
	length := 1
	// Subtract ubtract offsets less than 2^64, and start at a group count of 1 (5 bits).
	for i := uint(5); i < 64; i += 5 {
		var groupMax uint64 = 1 << i
		if n >= groupMax {
			n -= groupMax
			length++
		} else {
			break
		}
	}

	raw = append(raw, varintEncodeTable[byte(length+0x10-1)])
	var lsgBuf []byte
	for length > 0 {
		lsgBuf = append(lsgBuf, byte(n&0x1F))
		n >>= 5
		length--
	}
	for i := len(lsgBuf) - 1; i >= 0; i-- {
		raw = append(raw, varintEncodeTable[lsgBuf[i]])
	}

	return
}

func (v Varint) Encode() string {
	return string(v.EncodeBytes())
}

// DecodeBytes sets v to the value of raw, and returns the number of bytes consumed.
func (v *Varint) DecodeBytes(raw []byte) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("varint: no input")
	}

	length := int(varintDecodeTable[raw[0]])
	switch {
	case length >= 0x20:
		return 0, fmt.Errorf("varint: invalid length")
	case length < 0x10:
		*v = Varint(length)
		return 1, nil
	default:
		length -= 0x10 // Bring it down into range
		length += 1    // start counting at 1.
	}

	if len(raw) < length+1 {
		return 0, fmt.Errorf("varint: eof")
	}
	// Idea: getting the bit pattern right for uin64 overflow is hard.  Instead,
	// just compare the encoded form of uint64 max.
	if bytes.Compare(raw, []byte("xeyyyyyyyyyyyf")) > 0 {
		return 0, fmt.Errorf("varint: overflow")
	}

	var num uint64
	var offset uint64
	for i := uint(1); i <= uint(length); i++ {
		group := varintDecodeTable[raw[i]]
		if group == 0xFF {
			return 0, fmt.Errorf("varint: invalid coding")
		}
		offset += 1 << (5 * (i - 1))
		num = (num << 5) | uint64(group)
	}

	*v = Varint(num + offset + 0x10 - 1)

	return length + 1, nil
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
		return fmt.Errorf("excess input")
	}
	// Don't overwrite it on error
	*v = tmp
	return nil
}

func (v Varint) String() string {
	return fmt.Sprintf("%s(%d)", v.Encode(), v)
}

func buildCodingTables() (encoding, decoding []byte) {
	encoding = make([]byte, 256)
	for i := 0; i < len(encoding); i++ {
		encoding[i] = 0xFF
	}
	decoding = make([]byte, 256)
	for i := 0; i < len(decoding); i++ {
		decoding[i] = 0xFF
	}

	mapping := map[byte]byte{
		'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7,
		'8': 8, '9': 9, 'a': 10, 'b': 11, 'c': 12, 'd': 13, 'e': 14, 'f': 15,
		'g': 16, 'h': 17, 'j': 18, 'k': 19, 'm': 20, 'n': 21, 'p': 22, 'q': 23,
		'r': 24, 's': 25, 't': 26, 'v': 27, 'w': 28, 'x': 29, 'y': 30, 'z': 31,
	}

	for c, v := range mapping {
		decoding[int(c)] = v
		encoding[int(v)] = c
	}

	return
}
