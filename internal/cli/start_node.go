package cli

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/linoss-7/D7024E-Project/pkg/node"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(StartNodeCmd)
}

var StartNodeCmd = &cobra.Command{
	Use:   "start_node",
	Short: "Start a new node",
	Long:  "Start a new node in a UDP network",
	Run: func(cmd *cobra.Command, args []string) {
		net := network.NewUDPNetwork()
		port := args[0]
		iport, err := strconv.Atoi(port)
		if err != nil {
			cmd.Println("Invalid port number")
			return
		}
		addr := network.Address{
			IP:   "0.0.0.0",
			Port: iport,
		}
		newNode, err := node.NewNode(net, addr)
		if err != nil {
			cmd.Println("Failed to create node:", err)
			return
		}

		newNode.Handle("message", func(msg network.Message) error {
			return nil
		})

		waitSeconds := rand.Intn(21) + 10 // random int between 10 and 30
		newNode.Start()
		// Wait for random seconds, then send a message to next node (if any)
		time.AfterFunc(time.Duration(waitSeconds)*time.Second, func() {
			if iport < 8049 {
				nextAddr := network.Address{
					IP:   fmt.Sprintf("node_%d", (iport-8001)+1), // Adjust index as needed
					Port: iport + 1,
				}
				err := newNode.SendString(nextAddr, "message", "Hello from "+addr.String())
				if err != nil {
					cmd.Println("Failed to send message:", err)
					return
				}
			}
		})
		//cmd.Printf("Node started at %s. Listening for messages...\n", addr.String())

		select {} // Block forever, keeping the node alive

	},
}
