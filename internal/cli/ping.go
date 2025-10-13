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
	rootCmd.AddCommand(PingCmd)
}

var PingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the node",
	Long:  "Ping a Kademlia node in an UDP network",
	Run: func(cmd *cobra.Command, args []string) {
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

		// Send a ping rpc to the node

		resp, err := newNode.SendAndAwaitResponse("ping", network.Address{IP: info.IP, Port: info.Port}, common.DefaultKademliaMessage(*id, nil), 10.0)

		if err != nil {
			cmd.Println("Ping failed:", err)
			return
		}

		if resp == nil {
			cmd.Println("No response received")
			return
		}

		cmd.Println("Ping successful! Response RPC ID:", resp.RPCId, " from ", resp.SenderId)

	},
}
