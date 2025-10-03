package cli

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(PutCmd)
}

var PutCmd = &cobra.Command{
	Use:   "put",
	Short: "Put a value in the Kademlia network",
	Long:  "Put a value in the Kademlia network",
	Run: func(cmd *cobra.Command, args []string) {
		value := args[0]

		net := network.NewUDPNetwork()

		// Read the node info from the json file

		home, err := os.UserHomeDir()
		if err != nil {
			cmd.Println("Failed to get home directory:", err)
			return
		}

		jsonFolderPath := filepath.Join(home, ".kademlia", "nodes")
		file, err := os.Open(filepath.Join(jsonFolderPath, "node_info.json"))
		if err != nil {
			cmd.Println("Failed to open node file:", err)
			return
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		var info common.NodeInfo
		err = decoder.Decode(&info)
		if err != nil {
			cmd.Println("Failed to decode node info:", err)
			return
		}

		addr := network.Address{
			IP:   info.IP,
			Port: info.Port + 1,
		}

		// Create a new kademlia node to send and recieve rpcs
		id := utils.NewRandomBitArray(160)
		newNode, err := kademlia.NewKademliaNode(net, addr, *id, 4, 3)
		if err != nil {
			cmd.Println("Failed to create node:", err)
			return
		}

		// Send a put rpc to the node

		msg := common.DefaultKademliaMessage(*id, []byte(value))

		resp, err := newNode.SendAndAwaitResponse("put", network.Address{IP: info.IP, Port: info.Port}, msg)

		if err != nil {
			cmd.Println("Put failed:", err)
			return
		}

		if resp == nil {
			cmd.Println("No response received")
			return
		}

		// Check response, should contain the key for the value
		key := utils.NewBitArrayFromBytes(resp.Body, 160)
		hexKey := hex.EncodeToString(key.ToBytes())
		cmd.Println("Value stored with key:", hexKey)
	},
}
