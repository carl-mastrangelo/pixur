package schema

import (
	"testing"
)

func TestVarintEncodingZero(t *testing.T) {
	var num Varint = 0

	if text := num.Encode(); text != "g" {
		t.Fatalf("Expected %v but was %v", "g", text)
	}
}

func TestVarintEncodingLarge(t *testing.T) {
	var num Varint = 72374

	if text := num.Encode(); text != "m15mn" {
		t.Fatalf("Expected %v but was %v", "m15mn", text)
	}
}

func TestVarintEncodingNegative(t *testing.T) {
	var num Varint = -1

	if text := num.Encode(); text != "xeyyyyyyyyyyyy" {
		t.Fatalf("Expected %v but was %v", "xeyyyyyyyyyyyy", text)
	}
}

func TestVarintEncodingSingle(t *testing.T) {
	var num Varint = 0xF

	if text := num.Encode(); text != "he" {
		t.Fatalf("Expected %v but was %v", "he", text)
	}
}

func TestVarintDecodingZero(t *testing.T) {
	var num Varint = -1

	consumed, err := num.Decode("g")
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

	consumed, err := num.Decode("m15mn")
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

	consumed, err := num.Decode("xeyyyyyyyyyyyy")
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

	consumed, err := num.Decode("hz")
	if err != nil {
		t.Fatal(err)
	}
	if num != 32 {
		t.Fatalf("Expected %v but was %v", 32, num)
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

	consumed, err := num.Decode("hu")
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

func TestVarintDecodingFailOnTooLong(t *testing.T) {
	var num Varint = 3

	consumed, err := num.Decode("0123456789abcde")
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

	consumed, err := num.Decode("gg")
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
