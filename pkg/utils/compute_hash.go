package utils

import "crypto/sha256"

func ComputeHash(data string, length int) *BitArray {

	// Sha256 hash and truncate to desired length in bits
	hash := sha256.Sum256([]byte(data))

	return NewBitArrayFromBytes(hash[:], length)
}
