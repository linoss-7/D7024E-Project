package utils

import (
	"math/big"
	"math/rand"
	"strings"
)

// BitArray represents a resizable array of bits
type BitArray struct {
	bits   []byte
	length int // number of bits
}

// NewBitArray creates a new bit array with the given number of bits
func NewBitArray(length int) *BitArray {
	byteLen := (length + 7) / 8
	return &BitArray{
		bits:   make([]byte, byteLen),
		length: length,
	}
}

func NewRandomBitArray(length int) *BitArray {
	bitArray := NewBitArray(length)
	for i := 0; i < length; i++ {
		bitArray.Set(i, rand.Intn(2) == 1)
	}
	return bitArray
}

func NewBitArrayFromBytes(data []byte, length int) *BitArray {
	if len(data)*8 < length {
		panic("byte slice too short for the specified length")
	}
	return &BitArray{
		bits:   data[:byte(len(data))],
		length: length,
	}
}

// Set sets the bit at position i (0-based) to 1 or 0
func (b *BitArray) Set(i int, value bool) {
	if i < 0 || i >= b.length {
		panic("index out of range")
	}
	byteIndex := i / 8
	bitIndex := uint(i % 8)
	if value {
		b.bits[byteIndex] |= 1 << bitIndex
	} else {
		b.bits[byteIndex] &^= 1 << bitIndex
	}
}

// Get returns the value of the bit at position i
func (b *BitArray) Get(i int) bool {
	if i < 0 || i >= b.length {
		panic("index out of range")
	}
	byteIndex := i / 8
	bitIndex := uint(i % 8)
	return (b.bits[byteIndex] & (1 << bitIndex)) != 0
}

// Size returns the number of bits
func (b *BitArray) Size() int {
	return b.length
}

// ToBytes returns the underlying byte slice (truncated to actual size)
func (b *BitArray) ToBytes() []byte {
	return b.bits
}

// ToString returns the bit array as a string of "0" and "1"
func (b *BitArray) ToString() string {
	var sb strings.Builder
	for i := 0; i < b.length; i++ {
		if b.Get(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return sb.String()
}

func (b *BitArray) ToBigInt() *big.Int {
	res := new(big.Int)
	for i := 0; i < b.Size(); i++ {
		if b.Get(i) {
			bitPos := b.Size() - 1 - i // MSB = highest power
			res.SetBit(res, bitPos, 1)
		}
	}
	return res
}

func (b *BitArray) Xor(other BitArray) *BitArray {
	if b.Size() != other.Size() {
		panic("bit arrays must be of the same size to XOR")
	}

	result := NewBitArray(b.Size())
	for i := 0; i < b.Size(); i++ {
		result.Set(i, b.Get(i) != other.Get(i)) // XOR operation
	}
	return result
}

func (b *BitArray) Equals(other BitArray) bool {
	if b.Size() != other.Size() {
		return false
	}
	for i := 0; i < b.Size(); i++ {
		if b.Get(i) != other.Get(i) {
			return false
		}
	}
	return true
}
