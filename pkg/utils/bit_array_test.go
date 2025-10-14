package utils

import (
	"math/big"
	"testing"
)

func TestBitArray_SetGet(t *testing.T) {
	b := NewBitArray(8)
	b.Set(0, true)
	b.Set(7, true)
	if !b.Get(0) || !b.Get(7) {
		t.Errorf("Set/Get failed at edges")
	}
	b.Set(0, false)
	if b.Get(0) {
		t.Errorf("Set false failed")
	}
}

func TestBitArray_ToString(t *testing.T) {
	b := NewBitArray(4)
	b.Set(0, true)
	b.Set(2, true)
	s := b.ToString()
	if s != "1010" {
		t.Errorf("ToString got %s, want 1010", s)
	}
}

func TestBitArray_XorEquals(t *testing.T) {
	a := NewBitArray(4)
	b := NewBitArray(4)
	a.Set(0, true)
	b.Set(1, true)
	x := a.Xor(*b)
	if x.ToString() != "1100" {
		t.Errorf("Xor got %s, want 1100", x.ToString())
	}
	if a.Equals(*b) {
		t.Errorf("Equals should be false")
	}

	// Test different lengths
	c := NewBitArray(5)
	if a.Equals(*c) {
		t.Errorf("Equals should be false for different lengths")
	}
}

func TestBitArray_ToBigInt(t *testing.T) {
	b := NewBitArray(4)
	b.Set(0, true)
	b.Set(3, true)
	bi := b.ToBigInt()
	want := big.NewInt(9) // 1001
	if bi.Cmp(want) != 0 {
		t.Errorf("ToBigInt got %v, want %v", bi, want)
	}
}

func TestBitArray_Size(t *testing.T) {
	b := NewBitArray(10)
	if b.Size() != 10 {
		t.Errorf("Size got %d, want 10", b.Size())
	}

	if len(b.ToBytes()) != 2 { // 10 bits = 2 bytes
		t.Errorf("ToBytes length got %d, want 2", len(b.ToBytes()))
	}
}

func TestBitArray_RandomArrays(t *testing.T) {
	b := NewRandomBitArray(160)

	c := NewRandomBitArray(160)

	// Very small chance of collision, but still possible
	if b.Equals(*c) {
		t.Errorf("Two random BitArrays should not be equal")
	}
}

func TestBitArray_FromBytes(t *testing.T) {
	data := []byte{0b10101010, 0b11001100}
	b := NewBitArrayFromBytes(data, 16)
	if b.ToString() != "0101010100110011" { // LSB first
		t.Errorf("FromBytes got %s, want 0101010100110011", b.ToString())
	}

	// Back and forth from bytes

	b2 := NewBitArrayFromBytes(b.ToBytes(), 16)
	if !b.Equals(*b2) {
		t.Errorf("FromBytes/ToBytes roundtrip failed")
	}
}

func TestBitArray_FromString(t *testing.T) {
	b := NewBitArrayFromString("1011001")
	if b.Size() != 7 {
		t.Errorf("FromString size got %d, want 7", b.Size())
	}

	if b.ToString() != "1011001" {
		t.Errorf("FromString got %s, want 1011001", b.ToString())
	}
}

func TestFindClosestNodes(t *testing.T) {
	data1 := []byte{0b00000000, 0b10000000}
	data2 := []byte{0b00000000, 0b11111111}

	b1 := NewBitArrayFromBytes(data1, 16)
	if b1.ToString() != "0000000000000001" {
		t.Errorf("BitArray1 string got %s, want 0000000000000001", b1.ToString())
	}

	b2 := NewBitArrayFromBytes(data2, 16)
	if b2.ToString() != "0000000011111111" {
		t.Errorf("BitArray2 string got %s, want 0000000011111111", b2.ToString())
	}

	targetID := NewBitArrayFromBytes([]byte{0b00000000, 0b00000000}, 16)
	if targetID.ToString() != "0000000000000000" {
		t.Errorf("TargetID string got %s, want 0000000000000000", targetID.ToString())
	}

	if !b1.CloserTo(*targetID, *b2) {
		t.Errorf("b1 should be closer to targetID than b2")
	}
}

// Test that ToBigInt returns the unsigned magnitude (non-negative).
func TestBitArray_ToBigInt_Unsigned(t *testing.T) {
	// 4-bit array; set MSB (index 0) so the bit-string is "1000".
	b := NewBitArray(4)
	b.Set(0, true) // MSB
	// unsigned value should be 2^(4-1) = 8
	got := b.ToBigInt()
	want := big.NewInt(8)
	if got.Cmp(want) != 0 {
		t.Fatalf("ToBigInt unsigned: got %v, want %v", got, want)
	}
	if got.Sign() < 0 {
		t.Fatalf("ToBigInt returned negative for an unsigned magnitude: %v", got)
	}
}
