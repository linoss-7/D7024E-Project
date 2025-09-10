package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(SendMSGCmd)
}

var SendMSGCmd = &cobra.Command{
	Use:   "send_msg",
	Short: "Send a message to a node",
	Long:  "Send a message to a specific node in the network",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Sending message...")
	},
}
