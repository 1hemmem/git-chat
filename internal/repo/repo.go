package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2"
)

func GetGitHubUsername() (string, error) {
	stdOut, _, err := gh.Exec("api", "user", "-q", ".login")
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub username: %v", err)
	}
	return strings.TrimSpace(stdOut.String()), nil
}

func ResolveRepo(repo string) string {
	if strings.Contains(repo, "/") {
		return repo
	}
	username, err := GetGitHubUsername()
	if err != nil {
		return repo
	}
	return username + "/" + repo
}

func CachePath(repoFull string) string {
	parts := strings.SplitN(repoFull, "/", 2)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "git-chat", parts[0]+"-"+parts[1])
}

func CloneOrPull(repoFull, localPath string) error {
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
