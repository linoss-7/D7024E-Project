package cli

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/proto_gen"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
)

func init() {
	rootCmd.AddCommand(GetCmd)
}

var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a value from the Kademlia network",
	Long:  "Get a value from the Kademlia network",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Println("Usage: get <key>")
			return
		}

		key := args[0]

		// Expect the user to pass the hex-encoded key (as returned by put). Decode
		// it to the raw 160-bit (20-byte) representation used on the wire.
		decodedKey, err := hex.DecodeString(key)
		if err != nil {
			cmd.Println("Failed to decode key (expect hex string):", err)
			return
		}
		if len(decodedKey) != 20 {
			cmd.Println("Invalid key length: expected 40 hex chars representing 20 bytes (160 bits)")
			return
		}

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
		newNode, err := kademlia.NewKademliaNode(net, addr, *id, 4, 3, 10.0)
		if err != nil {
			cmd.Println("Failed to create node:", err)
			return
		}

		// Send a get rpc to the node

		msg := common.DefaultKademliaMessage(*id, decodedKey)

		resp, err := newNode.SendAndAwaitResponse("get", network.Address{IP: info.IP, Port: info.Port}, msg, 10.0)

		if err != nil {
			cmd.Println("Get failed:", err)
			return
		}

		if resp == nil {
			cmd.Println("No response received")
			return
		}

		// Check response, Unmarshal the ValueAndNodesMessage
		var vanm proto_gen.ValueAndNodesMessage
		err = proto.Unmarshal(resp.Body, &vanm)
		if err != nil {
			cmd.Println("Failed to unmarshal response:", err)
			return
		}

		value := string(vanm.Value)
		cmd.Println("Value retrieved for key:", value)
		cmd.Println("Found at nodes:")
		for _, node := range vanm.Nodes.Nodes {
			cmd.Printf("- %s:%d\n", node.IP, node.Port)
		}

		// Close the temporary node we created to free the port/socket
		newNode.Exit()
	},
}
