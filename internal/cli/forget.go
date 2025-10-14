package cli

import (
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
	rootCmd.AddCommand(ForgetCmd)
}

var ForgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Forget a key in the Kademlia network",
	Long:  "Forget a key in the Kademlia network",
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

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

		msg := common.DefaultKademliaMessage(*id, []byte(key))

		resp, err := newNode.SendAndAwaitResponse("forget", network.Address{IP: info.IP, Port: info.Port}, msg, 10.0)

		if err != nil {
			cmd.Println("Forget failed:", err)
			return
		}

		if resp == nil {
			cmd.Println("No response received")
			return
		}
	},
}
