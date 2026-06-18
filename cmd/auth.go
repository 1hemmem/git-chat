package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"git-chat/internal/auth"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage GitHub authentication",
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh [scopes...]",
	Short: "Refresh GitHub auth with additional scopes",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Refreshing auth with scopes: %s...\n", strings.Join(args, ", "))
		fmt.Println("Opening browser for authentication. Please complete the flow to continue.")
		if err := auth.RefreshAuthTerminal(args...); err != nil {
			return err
		}
		fmt.Println("Auth refreshed successfully.")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authRefreshCmd)
	RootCmd.AddCommand(authCmd)
}
