package cmd

import (
	"github.com/spf13/cobra"

	"git-chat/internal/auth"
	"git-chat/internal/tui"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a TUI for a specific group chat",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.EnsureScope("repo"); err != nil {
			return err
		}
		return tui.Run(args[0])
	},
}

func init() {
	RootCmd.AddCommand(openCmd)
}
