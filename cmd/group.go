package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2"
	"github.com/spf13/cobra"

	"git-chat/internal/auth"
	"git-chat/internal/repo"
)

const readmeTemplate = `# %s

This repository is a group chat managed by [git-chat](https://github.com/1hemmem/git-chat).

## Getting Started

1. Install git-chat:
   ` + "```" + `
   git clone https://github.com/1hemmem/git-chat.git
   cd git-chat && make install
   ` + "```" + `

2. Authenticate with GitHub:
   ` + "```" + `
   git-chat auth refresh
   ` + "```" + `

3. Open the live-chat TUI:
   ` + "```" + `
   git-chat open %s
   ` + "```" + `

`

var createGroupCmd = &cobra.Command{
	Use:   "creategroup <group_name>",
	Short: "Create a new private repository (group)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]
		if err := auth.EnsureScope("repo"); err != nil {
			return err
		}
		fmt.Printf("Creating repository: %s...\n", repoName)
		_, stdErr, err := gh.Exec("repo", "create", repoName, "--private")
		if err != nil {
			stderr := stdErr.String()
			if strings.Contains(stderr, "already exists") {
				return fmt.Errorf("repository %q already exists", repoName)
			}
			return fmt.Errorf("creating repository %q failed", repoName)
		}
		fmt.Println("Repository created successfully.")
		repoFull := repo.ResolveRepo(repoName)
		_, _, err = gh.Exec("repo", "edit", repoFull, "--add-topic", "chat-over-git-repo")
		if err != nil {
			return fmt.Errorf("tagging repository %q as a group chat failed", repoName)
		}
		fmt.Println("Group tagged successfully.")

		localPath := repo.CachePath(repoFull)
		if err := repo.EnsureCloned(repoFull, localPath); err != nil {
			return fmt.Errorf("failed to clone repository: %v", err)
		}

		readmePath := filepath.Join(localPath, "README.md")
		readmeContent := fmt.Sprintf(readmeTemplate, repoName, repoName)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0o644); err != nil {
			return fmt.Errorf("failed to write README.md: %v", err)
		}

		gitCmd := exec.Command("git", "-C", localPath, "add", "README.md")
		if _, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to stage README.md: %v", err)
		}

		gitCmd = exec.Command("git", "-C", localPath, "commit", "-m", "Initial commit with README.md")
		if _, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to commit README.md: %v", err)
		}

		gitCmd = exec.Command("git", "-C", localPath, "push", "origin", "main")
		if _, err := gitCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to push README.md: %v", err)
		}

		fmt.Println("Initial README.md committed and pushed.")
		return nil
	},
}

var deleteGroupCmd = &cobra.Command{
	Use:   "deletegroup <group_name>",
	Short: "Delete an existing repository (group)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]
		if err := auth.EnsureScope("delete_repo"); err != nil {
			return err
		}
		repoFull, err := repo.ResolveGroup(repoName)
		if err != nil {
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
		fmt.Printf("Deleting repository: %s...\n", repoFull)
		_, stdErr, err := gh.Exec("repo", "delete", repoFull, "--yes")
		if err != nil {
			stderr := stdErr.String()
			if strings.Contains(stderr, "not found") || strings.Contains(stderr, "Not Found") {
				return fmt.Errorf("repository %q not found — check the name or list groups with: git-chat listgroups", repoFull)
			}
			return fmt.Errorf("deleting repository %q failed", repoFull)
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
		if err := auth.EnsureScope("repo"); err != nil {
			return err
		}
		repoFull, err := repo.ResolveGroup(repoName)
		if err != nil {
			return err
		}
		parts := strings.SplitN(repoFull, "/", 2)
		owner := parts[0]
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

var listGroupsCmd = &cobra.Command{
	Use:   "listgroups",
	Short: "List all group chats you have access to",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.EnsureScope("repo"); err != nil {
			return err
		}
		stdOut, _, err := gh.Exec("api", "search/repositories?q=topic:chat-over-git-repo&per_page=100", "--jq", ".items[].full_name")
		if err != nil {
			return fmt.Errorf("listing groups failed")
		}
		output := strings.TrimSpace(stdOut.String())
		if output == "" {
			fmt.Println("No groups found.")
			return nil
		}
		fmt.Print(output)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(createGroupCmd)
	RootCmd.AddCommand(deleteGroupCmd)
	RootCmd.AddCommand(addMemberCmd)
	RootCmd.AddCommand(listGroupsCmd)
}
