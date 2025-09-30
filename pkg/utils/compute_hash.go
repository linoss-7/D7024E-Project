package utils

import "crypto/sha256"

func ComputeHash(data string, length int) *BitArray {

	// Sha256 hash and truncate to desired length in bits
	hash := sha256.Sum256([]byte(data))

	if length > len(hash) {
		length = len(hash)
	}
	return NewBitArrayFromBytes(hash[:length], length*8)
}
