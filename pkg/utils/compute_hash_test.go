package utils

import "testing"

func TestComputeHash_TestSameString(t *testing.T) {
	str := "Hello, World!"
	hash1 := ComputeHash(str, 160)
	hash2 := ComputeHash(str, 160)

	if !hash1.Equals(*hash2) {
		t.Errorf("Hashes for the same string do not match: %v vs %v", hash1, hash2)
	}
}

func TestComputeHash_TestDifferentStrings(t *testing.T) {
	str1 := "Hello, World!"
	str2 := "Goodbye, World!"
	hash1 := ComputeHash(str1, 160)
	hash2 := ComputeHash(str2, 160)

	if hash1.Equals(*hash2) {
		t.Errorf("Hashes for different strings should not match: %v vs %v", hash1, hash2)
	}
}

func TestComputHash_TestHashLength(t *testing.T) {
	str := "Hello, World!"
	lengths := []int{8, 16, 32, 64, 128, 160}
	for _, length := range lengths {
		hash := ComputeHash(str, length)
		if hash.Size() != length {
			t.Errorf("Hash length for %d bits is incorrect: got %d, want %d", length, hash.Size(), length)
		}
	}
}
