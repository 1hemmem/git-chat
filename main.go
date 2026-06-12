package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/spf13/cobra"
)

type Message struct {
	Author    string
	Timestamp string
	Body      string
}

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

func getGitHubUsername() (string, error) {
	stdOut, _, err := gh.Exec("api", "user", "-q", ".login")
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username: %v", err)
	}
	return strings.TrimSpace(stdOut.String()), nil
}

func resolveRepo(repo string) string {
	if strings.Contains(repo, "/") {
		return repo
	}
	username, err := getGitHubUsername()
	if err != nil {
		return repo
	}
	return username + "/" + repo
}

func cachePath(repoFull string) string {
	parts := strings.SplitN(repoFull, "/", 2)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "git-chat", parts[0]+"-"+parts[1])
}

func cloneOrPull(repoFull, localPath string) error {
	if _, err := os.Stat(filepath.Join(localPath, ".git")); os.IsNotExist(err) {
		cmd := exec.Command("gh", "repo", "clone", repoFull, localPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	cmd := exec.Command("git", "-C", localPath, "fetch", "origin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %v", err)
	}
	cmd = exec.Command("git", "-C", localPath, "checkout", "main")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	cmd = exec.Command("git", "-C", localPath, "reset", "--hard", "origin/main")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func generateFilename(username string) string {
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	b := make([]byte, 3)
	rand.Read(b)
	suffix := hex.EncodeToString(b)
	return timestamp + "_" + username + "_" + suffix + ".txt"
}

func parseFilename(name string) (author, timestamp string, err error) {
	base := strings.TrimSuffix(name, ".txt")
	parts := strings.SplitN(base, "_", 3)
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid message filename: %s", name)
	}
	return parts[1], parts[0], nil
}

func sendMessage(repo, body string) error {
	repoFull := resolveRepo(repo)
	username, err := getGitHubUsername()
	if err != nil {
		return err
	}
	localPath := cachePath(repoFull)
	if err := cloneOrPull(repoFull, localPath); err != nil {
		return fmt.Errorf("failed to clone/pull repository: %v", err)
	}
	msgsDir := filepath.Join(localPath, "messages")
	if err := os.MkdirAll(msgsDir, 0o755); err != nil {
		return fmt.Errorf("failed to create messages directory: %v", err)
	}
	filename := generateFilename(username)
	filePath := filepath.Join(msgsDir, filename)
	if err := os.WriteFile(filePath, []byte(body), 0o644); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}
	cmd := exec.Command("git", "-C", localPath, "add", "messages/"+filename)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", localPath, "commit", "-m", fmt.Sprintf("message from %s", username))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", localPath, "push", "origin", "main")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %v\n%s", err, out)
	}
	return nil
}

func readMessages(repo string) ([]Message, error) {
	repoFull := resolveRepo(repo)
	localPath := cachePath(repoFull)
	if err := cloneOrPull(repoFull, localPath); err != nil {
		return nil, fmt.Errorf("failed to clone/pull repository: %v", err)
	}
	msgsDir := filepath.Join(localPath, "messages")
	entries, err := os.ReadDir(msgsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no messages yet in this group")
		}
		return nil, fmt.Errorf("failed to read messages: %v", err)
	}
	var filenames []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
			filenames = append(filenames, e.Name())
		}
	}
	sort.Strings(filenames)
	var messages []Message
	for _, name := range filenames {
		author, ts, err := parseFilename(name)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(msgsDir, name))
		if err != nil {
			continue
		}
		messages = append(messages, Message{
			Author:    author,
			Timestamp: ts,
			Body:      strings.TrimSpace(string(content)),
		})
	}
	return messages, nil
}

var rootCmd = &cobra.Command{
	Use:   "git-chat",
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
		_, stdErr, err := gh.Exec("repo", "create", repoName, "--private")
		if err != nil {
			log.Fatalf("Error creating repo: %v\nStderr: %s", err, stdErr.String())
		}

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
		_, stdErr, err := gh.Exec("repo", "delete", repoName, "--yes")
		if err != nil {
			log.Fatalf("Error deleting repo: %v\nStderr: %s", err, stdErr.String())
		}

		fmt.Println("Repository deleted successfully.")
		return nil
	},
}

var addMemberCmd = &cobra.Command{
	Use:   "addmember <group_name> <username>",
	Short: "Add a member to a repository (group)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]
		username := args[1]
		if err := ensureScope("repo"); err != nil {
			return err
		}
		owner, err := getGitHubUsername()
		if err != nil {
			return err
		}
		fmt.Printf("Adding %s to %s...\n", username, repoName)
		path := fmt.Sprintf("repos/%s/%s/collaborators/%s", owner, repoName, username)
		_, _, err = gh.Exec("api", path, "-X", "PUT", "-f", "permission=push")
		if err != nil {
			return fmt.Errorf("failed to add member: %v", err)
		}
		fmt.Printf("Added %s to %s as a collaborator.\n", username, repoName)
		return nil
	},
}

var sendmsgCmd = &cobra.Command{
	Use:   "sendmsg <group_name> <message>",
	Short: "Send a message to a group",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]
		message := strings.Join(args[1:], " ")
		if err := ensureScope("repo"); err != nil {
			return err
		}
		if err := sendMessage(repo, message); err != nil {
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
		if err := ensureScope("repo"); err != nil {
			return err
		}
		messages, err := readMessages(args[0])
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
	rootCmd.AddCommand(addMemberCmd)
	rootCmd.AddCommand(sendmsgCmd)
	rootCmd.AddCommand(readmsgsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
