package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/rpc_handlers"
	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/utils"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(StartNodeCmd)
}

var StartNodeCmd = &cobra.Command{
	Use:   "start_node",
	Short: "Start a new node",
	Long:  "Start a new kademlia node in an UDP network",
	Run: func(cmd *cobra.Command, args []string) {

		k := 4
		alpha := 3
		net := network.NewUDPNetwork()
		port := args[0]
		iport, err := strconv.Atoi(port)
		if err != nil {
			cmd.Println("Invalid port number")
			return
		}

		cmd.Println("Starting node on port", iport)
		addr := network.Address{
			IP:   "node_" + strconv.Itoa(iport-8001),
			Port: iport,
		}

		id := utils.NewRandomBitArray(160)

		newNode, err := kademlia.NewKademliaNode(net, addr, *id, k, alpha)
		if err != nil {
			cmd.Println("Failed to create node:", err)
			return
		}

		// Register ping and find_node handlers

		pingHandler := rpc_handlers.NewPingHandler(newNode.Node, newNode.ID)
		findNodeHandler := rpc_handlers.NewFindNodeHandler(newNode, newNode.RoutingTable)

		// Register handlers to the node in the kademlia_node struct

		newNode.Node.Handle("ping", pingHandler.Handle)
		newNode.Node.Handle("find_node", findNodeHandler.Handle)

		// Save the ip and port to json file so commands know where to send

		go func() {
			home, err := os.UserHomeDir()
			if err != nil {
				cmd.Println("Failed to get home directory:", err)
				return
			}

			jsonFolderPath := filepath.Join(home, ".kademlia", "nodes")
			err = os.MkdirAll(jsonFolderPath, 0755)
			if err != nil {
				cmd.Println("Failed to create .kademlia directory:", err)
				return
			}
			file, err := os.Create(filepath.Join(jsonFolderPath, "node_info.json"))
			if err != nil {
				cmd.Println("Failed to create node file:", err)
				return
			}
			defer file.Close()

			encoder := json.NewEncoder(file)
			nodeInfo := common.NodeInfo{
				ID:   newNode.ID,
				IP:   addr.IP,
				Port: addr.Port,
			}

			err = encoder.Encode(nodeInfo)
			if err != nil {
				cmd.Println("Failed to encode node info:", err)
				return
			}
		}()

		select {} // Block forever, keeping the node alive

	},
}
