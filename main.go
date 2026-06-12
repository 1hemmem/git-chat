package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/spf13/cobra"
)

func hasScope(scope string) bool {
	stdOut, _, err := gh.Exec("auth", "status")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(stdOut.String(), "\n") {
		if strings.Contains(line, "Token scopes:") {
			return strings.Contains(line, scope)
		}
	}
	return false
}

func isAuthenticated() bool {
	_, _, err := gh.Exec("auth", "status")
	return err == nil
}

func refreshAuthTerminal(scopes ...string) error {
	args := []string{"auth", "refresh", "-h", "github.com"}
	for _, s := range scopes {
		args = append(args, "-s", s)
	}
	cmd := exec.Command("gh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to refresh auth: %v", err)
	}
	return nil
}

func ensureScope(scope string) error {
	if !isAuthenticated() {
		fmt.Println("You are not logged into GitHub.")
		fmt.Print("Would you like to log in now? This will open your browser. [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			return fmt.Errorf("not authenticated. Run:\n  gh auth login -s repo -s delete_repo")
		}
		fmt.Println("Opening browser for authentication. Please complete the flow to continue.")
		cmd := exec.Command("gh", "auth", "login", "-s", "repo", "-s", "delete_repo")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to log in: %v", err)
		}
		fmt.Println("Logged in successfully.")
		return nil
	}
	if hasScope(scope) {
		return nil
	}
	fmt.Printf("The '%s' scope is required but not found in your GitHub token.\n", scope)
	fmt.Print("Would you like to add it now? This will open your browser. [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %v", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		return fmt.Errorf("required scope '%s' not granted. Run:\n  gh auth refresh -h github.com -s %s", scope, scope)
	}
	fmt.Println("Refreshing GitHub auth...")
	fmt.Println("Opening browser for authentication. Please complete the flow to continue.")
	if err := refreshAuthTerminal(scope); err != nil {
		return err
	}
	fmt.Println("Auth refreshed successfully.")
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "git-whatsapp",
	Short: "A CLI tool for GitHub repo management",
}

var createGroupCmd = &cobra.Command{
	Use:   "creategroup <group_name>",
	Short: "Create a new private repository (group)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]
		if err := ensureScope("repo"); err != nil {
			return err
		}
		fmt.Printf("Creating repository: %s...\n", repoName)
		stdOut, stdErr, err := gh.Exec("repo", "create", repoName, "--private")
		if err != nil {
			log.Fatalf("Error creating repo: %v\nStderr: %s", err, stdErr.String())
		}
		fmt.Print(stdOut.String())

		fmt.Println("Repository created successfully.")
		return nil
	},
}

var deleteGroupCmd = &cobra.Command{
	Use:   "deletegroup <group_name>",
	Short: "Delete an existing repository (group)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]
		if err := ensureScope("delete_repo"); err != nil {
			return err
		}
		fmt.Printf("Are you sure you want to delete this group : '%s'? This action cannot be undone. [y/N]: ", repoName)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %v", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
		fmt.Printf("Deleting repository: %s...\n", repoName)
		stdOut, stdErr, err := gh.Exec("repo", "delete", repoName, "--yes")
		if err != nil {
			log.Fatalf("Error deleting repo: %v\nStderr: %s", err, stdErr.String())
		}
		fmt.Print(stdOut.String())

		fmt.Println("Repository deleted successfully.")
		return nil
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage GitHub authentication",
}

var authRefreshCmd = &cobra.Command{
	Use:   "refresh [scopes...]",
	Short: "Refresh GitHub auth with additional scopes",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Refreshing auth with scopes: %s...\n", strings.Join(args, ", "))
		fmt.Println("Opening browser for authentication. Please complete the flow to continue.")
		if err := refreshAuthTerminal(args...); err != nil {
			return err
		}
		fmt.Println("Auth refreshed successfully.")
		return nil
	},
}

func main() {
	authCmd.AddCommand(authRefreshCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(createGroupCmd)
	rootCmd.AddCommand(deleteGroupCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
