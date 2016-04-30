package schema

import (
	"strings"
	"testing"
)

func TestVarintEncodingZero(t *testing.T) {
	var num Varint = 0

	if text := num.Encode(); text != "0" {
		t.Fatalf("Expected %v but was %v", "0", text)
	}
}

func TestVarintEncodingLarge(t *testing.T) {
	var num Varint = 72374

	if text := num.Encode(); text != "k15m6" {
		t.Fatalf("Expected %v but was %v", "k15m6", text)
	}
}

func TestVarintString(t *testing.T) {
	var num Varint = 72374

	if text := num.String(); text != "k15m6(72374)" {
		t.Fatalf("Expected %v but was %v", "k15m6(72374)", text)
	}
}

func TestVarintEncodingNegative(t *testing.T) {
	var num Varint = -1

	if text := num.Encode(); text != "weyyyyyyyyyyyf" {
		t.Fatalf("Expected %v but was %v", "weyyyyyyyyyyyf", text)
	}
}

func TestVarintEncodingSingle(t *testing.T) {
	var num Varint = 0x10

	if text := num.Encode(); text != "g0" {
		t.Fatalf("Expected %v but was %v", "g0", text)
	}
}

func TestVarintDecodingZero(t *testing.T) {
	var num Varint = -1

	consumed, err := num.Decode("0")
	if err != nil {
		t.Fatal(err)
	}
	if num != 0 {
		t.Fatalf("Expected %v but was %v", 0, num)
	}
	if consumed != 1 {
		t.Fatal("not all bytes consumed")
	}
}

func TestVarintDecodingLarge(t *testing.T) {
	var num Varint

	consumed, err := num.Decode("k15m6")
	if err != nil {
		t.Fatal(err)
	}
	if num != 72374 {
		t.Fatalf("Expected %v but was %v", 72374, num)
	}
	if consumed != 5 {
		t.Fatal("not all bytes consumed")
	}
}

func TestVarintDecodingNegative(t *testing.T) {
	var num Varint

	consumed, err := num.Decode("weyyyyyyyyyyyf")
	if err != nil {
		t.Fatal(err)
	}
	if num != -1 {
		t.Fatalf("Expected %v but was %v", -1, num)
	}
	if consumed != 14 {
		t.Fatal("not all bytes consumed")
	}
}

func TestVarintDecodingSingle(t *testing.T) {
	var num Varint

	consumed, err := num.Decode("g0")
	if err != nil {
		t.Fatal(err)
	}
	if num != 16 {
		t.Fatalf("Expected %v but was %v", 16, num)
	}
	if consumed != 2 {
		t.Fatal("not all bytes consumed")
	}
}

func TestVarintDecodingMatchCase(t *testing.T) {
	var num Varint

	consumed, err := num.Decode("HZ")
	if err == nil {
		t.Fatal("expected an error")
	}
	if consumed != 0 {
		t.Fatalf("Expected no bytes to be consumed")
	}
	if num != 0 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestVarintDecodingFailOnInvalidInput(t *testing.T) {
	var num Varint

	consumed, err := num.Decode("gu")
	if err == nil {
		t.Fatal("expected an error")
	}
	if consumed != 0 {
		t.Fatalf("Expected no bytes to be consumed")
	}
	if num != 0 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestVarintDecodingFailOnEmpty(t *testing.T) {
	var num Varint = 3

	consumed, err := num.Decode("")
	if err == nil {
		t.Fatal("expected an error")
	}
	if consumed != 0 {
		t.Fatalf("Expected no bytes to be consumed")
	}
	if num != 3 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestVarintDecodingFailOnTooShort(t *testing.T) {
	var num Varint = 3

	consumed, err := num.Decode("h")
	if err == nil {
		t.Fatal("expected an error")
	}
	if consumed != 0 {
		t.Fatalf("Expected no bytes to be consumed")
	}
	if num != 3 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestVarintDecodingSucceedsOnExcess(t *testing.T) {
	var num Varint = 3

	consumed, err := num.Decode("00")
	if err != nil {
		t.Fatal(err)
	}
	if num != 0 {
		t.Fatalf("Expected %v but was %v", 0, num)
	}
	if consumed != 1 {
		t.Fatal("wrong number of bytes consumed")
	}
}

func TestVarintDecodeAllSucceeds(t *testing.T) {
	var num Varint = 0

	err := num.DecodeAll("F")
	if err != nil {
		t.Fatal(err)
	}
	if num != 0xF {
		t.Fatalf("Expected %v but was %v", 0xF, num)
	}
}

func TestVarintDecodeAllFailsOnBadInput(t *testing.T) {
	var num Varint = 0

	err := num.DecodeAll("G")
	if err == nil {
		t.Fatal("Expected error")
	}
}

func TestVarintDecodeAllFailsOnExcessInput(t *testing.T) {
	var num Varint = 0

	err := num.DecodeAll("11")
	if err != errExcessInput {
		t.Fatal("Expected error")
	}
	if num != 0 {
		t.Fatal("Should not have touched num")
	}
}

func TestDecodeFailsOnOverflow(t *testing.T) {
	v := new(Varint)
	_, err := v.Decode("y00b00es00000000")
	if err == nil {
		t.Fatal("expected overflow")
	}
}

func TestDecodeFailsOnOverflowEdge(t *testing.T) {
	v := new(Varint)
	_, err := v.Decode("weyyyyyyyyyyyg")
	if err == nil {
		t.Fatal("expected overflow")
	}
}

func TestDecodeFailsOnEmpty(t *testing.T) {
	v := new(Varint)
	_, err := v.DecodeBytes(nil)
	if err != errNoInput {
		t.Fatal("expected error")
	}
}

func TestDecodeFailsOnBadLength(t *testing.T) {
	v := new(Varint)
	_, err := v.Decode("u")
	if err != errInvalidLength {
		t.Fatal("expected error")
	}
}

func TestDecodeFailsOnOverflowUppercase(t *testing.T) {
	v := new(Varint)
	_, err := v.Decode("Y00b00es00000000")
	if err == nil {
		t.Fatal("expected overflow")
	}
}

func TestRoundTripLowers(t *testing.T) {
	cases := []string{
		"A",
	}
	for _, data := range cases {
		v := new(Varint)
		n, err := v.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		out := v.Encode()
		if strings.Compare(out, strings.ToLower(data[:n])) != 0 {
			t.Log("mismatch! ", out, strings.ToLower(data[:n]))
			t.Fail()
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		num := Varint(i)
		num.Encode()
	}
}

func BenchmarkRoundTrip(b *testing.B) {
	var num Varint

	buf := make([]byte, 0, 32)
	for i := 0; i < b.N; i++ {
		var total int
		for k := 0; k < 1<<16; k++ {
			num = Varint(k)
			buf = num.Append(buf)
			num.DecodeBytes(buf)
			total += int(num)
			buf = buf[:0]
		}
		if total != 65535*32768 {
			b.Fatal("bad encode")
		}
	}
}
