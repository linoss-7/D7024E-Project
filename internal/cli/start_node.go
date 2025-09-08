package cli

import (
	"strconv"

	"github.com/linoss-7/D7024E-Project/pkg/network"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(StartNodeCmd)
}

var StartNodeCmd = &cobra.Command{
	Use:   "start_node",
	Short: "Start a new node",
	Long:  "Start a new node in the gossip network",
	Run: func(cmd *cobra.Command, args []string) {
		net := network.NewUDPNetwork()
		port := args[0]
		iport, err := strconv.Atoi(port)
		if err != nil {
			cmd.Println("Invalid port number")
			return
		}
		addr := network.Address{
			IP:   "localhost",
			Port: iport,
		}
		net.Listen(addr)
	},
}
