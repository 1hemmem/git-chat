package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "git-chat",
	Short: "A CLI tool for GitHub repo management",
}
