package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/kademlia"
	"github.com/linoss-7/D7024E-Project/pkg/kademlia/common"
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

		newNode, err := kademlia.NewKademliaNode(net, addr, *id, k, alpha, 3600)
		if err != nil {
			cmd.Println("Failed to create node:", err)
			return
		}
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

		// Wait 30 seconds then attempt to join the network via the next node
		if iport != 8000 {
			bootstrapAddr := network.Address{
				IP:   "node_" + strconv.Itoa(iport-8002),
				Port: iport - 1,
			}

			bootstrapInfo := common.NodeInfo{
				IP:   bootstrapAddr.IP,
				Port: bootstrapAddr.Port,
			}

			go func() {
				time.Sleep(30 * time.Second)
				err := newNode.Join(bootstrapInfo)
				if err != nil {
					cmd.Println("Failed to join network:", err)
					return
				}
				cmd.Println("Successfully joined the network via", bootstrapAddr)
			}()
		}

		select {} // Block forever, keeping the node alive

	},
}
