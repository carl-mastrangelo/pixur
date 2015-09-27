package schema

import (
	"testing"
)

func TestBase32EncodingZero(t *testing.T) {
	var num B32Varint = 0

	data, err := num.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if text != "0" {
		t.Fatalf("Expected %v but was %v", "0", text)
	}
}

func TestBase32EncodingLarge(t *testing.T) {
	var num B32Varint = 72374

	data, err := num.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if text != "hhtv6" {
		t.Fatalf("Expected %v but was %v", "hhtv6", text)
	}
}

func TestBase32EncodingNegative(t *testing.T) {
	var num B32Varint = -1

	data, err := num.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if text != "zzzzzzzzzzzzzzzf" {
		t.Fatalf("Expected %v but was %v", "zzzzzzzzzzzzzzzf", text)
	}
}

func TestBase32EncodingSingle(t *testing.T) {
	var num B32Varint = 0xF

	data, err := num.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if text != "f" {
		t.Fatalf("Expected %v but was %v", "f", text)
	}
}

func TestBase32DecodingZero(t *testing.T) {
	var num B32Varint = -1

	err := num.UnmarshalText([]byte("0"))
	if err != nil {
		t.Fatal(err)
	}
	if num != 0 {
		t.Fatalf("Expected %v but was %v", 0, num)
	}
}

func TestBase32DecodingLarge(t *testing.T) {
	var num B32Varint

	err := num.UnmarshalText([]byte("hhtv6"))
	if err != nil {
		t.Fatal(err)
	}
	if num != 72374 {
		t.Fatalf("Expected %v but was %v", 72374, num)
	}
}

func TestBase32DecodingNegative(t *testing.T) {
	var num B32Varint

	err := num.UnmarshalText([]byte("zzzzzzzzzzzzzzzf"))
	if err != nil {
		t.Fatal(err)
	}
	if num != -1 {
		t.Fatalf("Expected %v but was %v", -1, num)
	}
}

func TestBase32DecodingSingle(t *testing.T) {
	var num B32Varint

	err := num.UnmarshalText([]byte("f"))
	if err != nil {
		t.Fatal(err)
	}
	if num != 15 {
		t.Fatalf("Expected %v but was %v", 15, num)
	}
}

func TestBase32DecodingIgnoreCase(t *testing.T) {
	var upper B32Varint
	var lower B32Varint

	err := upper.UnmarshalText([]byte("Za"))
	if err != nil {
		t.Fatal(err)
	}
	err = lower.UnmarshalText([]byte("zA"))
	if err != nil {
		t.Fatal(err)
	}
	if upper != lower {
		t.Fatalf("Expected %v = %v", upper, lower)
	}
}

func TestBase32DecodingIgnoreNumeric(t *testing.T) {
	var left B32Varint
	var right B32Varint

	err := left.UnmarshalText([]byte("L"))
	if err != nil {
		t.Fatal(err)
	}
	err = right.UnmarshalText([]byte("1"))
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("Expected %v = %v", left, right)
	}
}

func TestBase32DecodingFailOnInvalidInput(t *testing.T) {
	var num B32Varint = 3
	err := num.UnmarshalText([]byte("u"))
	if err == nil {
		t.Fatal("Expected an Error")
	}
	if num != 3 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestBase32DecodingFailOnEmpty(t *testing.T) {
	var num B32Varint = 3
	err := num.UnmarshalText([]byte{})
	if err == nil {
		t.Fatal("Expected an Error")
	}
	if num != 3 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestBase32DecodingFailOnTooLong(t *testing.T) {
	var num B32Varint = 3
	err := num.UnmarshalText([]byte("00000000000000001"))
	if err == nil {
		t.Fatal("Expected an Error")
	}
	if num != 3 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestBase32DecodingFailOnOverLong(t *testing.T) {
	var num B32Varint = 3
	err := num.UnmarshalText([]byte("g0"))
	if err == nil {
		t.Fatal("Expected an Error")
	}
	if num != 3 {
		t.Fatal("Should not have changed on bad input")
	}
}

func TestBase32DecodingIgnoreExcess(t *testing.T) {
	var num B32Varint = 1
	err := num.UnmarshalText([]byte("1!@#$%"))
	if err != nil {
		t.Fatal(err)
	}
	if num != 1 {
		t.Fatalf("Expected %v but was %v", 1, num)
	}
}
