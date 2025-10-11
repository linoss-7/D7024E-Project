package kademlia

import (
	"encoding/hex"
	"encoding/json"
	"net/http"

	//go get github.com/gin-gonic/gin
	"github.com/linoss-7/D7024E-Project/pkg/utils"
)

type KademliaRESTAPI struct {
	kademliaNode *KademliaNode
}

func NewKademliaRESTAPI(kademliaNode *KademliaNode) *KademliaRESTAPI {
	return &KademliaRESTAPI{
		kademliaNode: kademliaNode,
	}
}

func (api *KademliaRESTAPI) RegisterRoutes() {
	// Register API routes for put, get, forget and exit

	http.HandleFunc("/objects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Handle GET request
			idHex := r.URL.Query().Get("id")
			if idHex == "" {
				http.Error(w, "Missing id parameter", http.StatusBadRequest)
				return
			}
			idBytes, err := hex.DecodeString(idHex)
			if err != nil {
				http.Error(w, "Invalid id parameter", http.StatusBadRequest)
				return
			}
			id := utils.NewBitArrayFromBytes(idBytes, 160)

			dataObjects, _, err := api.kademliaNode.FindValueInNetwork(id)
			if err != nil {
				http.Error(w, "Failed to find value", http.StatusInternalServerError)
				return
			}

			// Return the found data objects
			w.Header().Set("Location", "/objects/"+idHex)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(dataObjects)
		case http.MethodPost:
			// Handle POST request
			var requestData struct {
				Data string `json:"data"`
			}
			err := json.NewDecoder(r.Body).Decode(&requestData)
			if err != nil {
				http.Error(w, "Invalid request payload", http.StatusBadRequest)
				return
			}

			// Check that the data has a Data field
			if requestData.Data == "" {
				http.Error(w, "Missing data field", http.StatusBadRequest)
				return
			}

			dataKey, err := api.kademliaNode.StoreInNetwork(requestData.Data)
			if err != nil {
				http.Error(w, "Failed to store data", http.StatusInternalServerError)
				return
			}

			hexKey := hex.EncodeToString(dataKey.ToBytes())

			w.Header().Set("Location", "/objects/"+hexKey)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"value": requestData.Data})
		case http.MethodDelete:
			// Handle DELETE request, forget value if node is refreshing it
			idHex := r.URL.Query().Get("id")
			if idHex == "" {
				http.Error(w, "Missing id parameter", http.StatusBadRequest)
				return
			}
			idBytes, err := hex.DecodeString(idHex)
			if err != nil {
				http.Error(w, "Invalid id parameter", http.StatusBadRequest)
				return
			}
			id := utils.NewBitArrayFromBytes(idBytes, 160)

			err = api.kademliaNode.Forget(id)
			if err != nil {
				http.Error(w, "Failed to forget value", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		api.kademliaNode.Node.Close()
		w.WriteHeader(http.StatusOK)
	})
}

// Start the REST API server
func (api *KademliaRESTAPI) Start(port string) error {
	return http.ListenAndServe(":"+port, nil)
}
