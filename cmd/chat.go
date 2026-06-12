package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"git-chat/internal/auth"
	"git-chat/internal/chat"
)

var sendmsgCmd = &cobra.Command{
	Use:   "sendmsg <group_name> <message>",
	Short: "Send a message to a group",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		message := strings.Join(args[1:], " ")
		if err := auth.EnsureScope("repo"); err != nil {
			return err
		}
		if err := chat.SendMessage(repo, message); err != nil {
			return err
		}
		fmt.Println("Message sent.")
		return nil
	},
}

var readmsgsCmd = &cobra.Command{
	Use:   "readmsgs <group_name>",
	Short: "Read messages from a group",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.EnsureScope("repo"); err != nil {
			return err
		}
		messages, err := chat.ReadMessages(args[0])
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			fmt.Println("No messages yet.")
			return nil
		}
		for _, msg := range messages {
			t, err := time.Parse("20060102T150405Z", msg.Timestamp)
			displayTime := msg.Timestamp
			if err == nil {
				displayTime = t.Local().Format("2006-01-02 15:04")
			}
			fmt.Printf("[%s] %s: %s\n", displayTime, msg.Author, msg.Body)
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(sendmsgCmd)
	RootCmd.AddCommand(readmsgsCmd)
}
