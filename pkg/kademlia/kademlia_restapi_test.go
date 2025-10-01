package kademlia

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

func TestPostObject(t *testing.T) {
	// return immediately, remove when implemented
	return
	net := network.NewMockNetwork(0.0)

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	// Create a kademlia node
	kademliaNode, err := NewKademliaNode(net, network.Address{IP: selfInfo.IP, Port: selfInfo.Port}, *utils.NewRandomBitArray(160), 20, 3)
	if err != nil {
		t.Fatalf("Failed to create kademlia node: %v", err)
	}

	// Create a REST API for the kademlia node
	api := NewKademliaRESTAPI(kademliaNode)
	api.RegisterRoutes()

	// Start a test server
	w := httptest.NewRecorder()

	req := httptest.NewRequest("POST", "/objects", strings.NewReader(`{"Data":"test"}`))
	req.Header.Set("Content-Type", "application/json")

	http.DefaultServeMux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	// Verify that the key is the hash of the data
	expectedHash := utils.ComputeHash("test", 160)
	expectedHex := hex.EncodeToString(expectedHash.ToBytes())

	// Body should contain the value

	// Check that the Location header contains the correct key
	if loc := w.Header().Get("Location"); loc != "/objects/"+expectedHex {
		t.Errorf("expected Location header to be /objects/%s, got %s", expectedHex, loc)
	}

	// Check that the response body contains the correct value
	expectedBody := `{"value":"test"}` + "\n"
	if w.Body.String() != expectedBody {
		t.Errorf("expected body to be %q, got %q", expectedBody, w.Body.String())
	}

	// Send exit request to clean up
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/exit", nil)
	http.DefaultServeMux.ServeHTTP(w, req)
	// Ensure we get a 200 OK response
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetObject(t *testing.T) {
	// return immediately, remove when implemented
	return
	net := network.NewMockNetwork(0.0)

	selfInfo := common.NodeInfo{
		ID:   *utils.NewRandomBitArray(160),
		IP:   "localhost",
		Port: 8000,
	}

	// Create a kademlia node
	kademliaNode, err := NewKademliaNode(net, network.Address{IP: selfInfo.IP, Port: selfInfo.Port}, *utils.NewRandomBitArray(160), 20, 3)
	if err != nil {
		t.Fatalf("Failed to create kademlia node: %v", err)
	}

	// Create a REST API for the kademlia node
	api := NewKademliaRESTAPI(kademliaNode)
	api.RegisterRoutes()

	// Store an object in the kademlia node
	dataKey, err := kademliaNode.StoreInNetwork("test")
	if err != nil {
		t.Fatalf("Failed to store object in kademlia node: %v", err)
	}

	// Start a test server
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/objects?id="+hex.EncodeToString(dataKey.ToBytes()), nil)
	http.DefaultServeMux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Check that the response body contains the correct value
	expectedBody := `[{"Data":"test"}]` + "\n"
	if w.Body.String() != expectedBody {
		t.Errorf("expected body to be %q, got %q", expectedBody, w.Body.String())
	}

	// Send exit request to clean up
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/exit", nil)
	http.DefaultServeMux.ServeHTTP(w, req)
	// Ensure we get a 200 OK response
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
