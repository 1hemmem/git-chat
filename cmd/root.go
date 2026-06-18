package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:          "git-chat",
	Short:        "A CLI + TUI for creating group chats backed by git and github.",
	SilenceUsage: true,
}
