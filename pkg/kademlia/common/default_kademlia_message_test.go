package common

import (
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

// Test for the NewKademliaMessage function for better code coverage :)

func TestKademliaMessage_NewKademliaMessage(t *testing.T) {
	SenderId := utils.NewRandomBitArray(160)
	Body := []byte("Test body")

	// Make 10 different messages to ensure different RPC IDs are generated
	prevId := utils.NewBitArray(160) // Empty ID to start with
	foundDifferent := false
	for i := 0; i < 10; i++ {
		msg := DefaultKademliaMessage(*SenderId, Body)

		// Check if the RPC id is the same as previous
		if !utils.NewBitArrayFromBytes(msg.RPCId, 160).Equals(*prevId) {
			foundDifferent = true
			break
		}
	}

	if !foundDifferent {
		t.Errorf("Generated RPC IDs are not unique!")
	}
}
