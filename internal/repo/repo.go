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

func ResolveGroup(repoName string) (string, error) {
	if strings.Contains(repoName, "/") {
		return repoName, nil
	}
	query := fmt.Sprintf("search/repositories?q=topic:chat-over-git-repo+%s+in:name&per_page=10", repoName)
	stdOut, _, err := gh.Exec("api", query, "--jq", ".items[].full_name")
	if err != nil {
		return "", fmt.Errorf("failed to search for group %q", repoName)
	}
	matches := strings.Fields(strings.TrimSpace(stdOut.String()))
	for _, m := range matches {
		parts := strings.SplitN(m, "/", 2)
		if len(parts) == 2 && parts[1] == repoName {
			return m, nil
		}
	}
	return "", fmt.Errorf("group %q not found\nRun 'git-chat listgroups' to see available groups", repoName)
}

func CachePath(repoFull string) string {
	parts := strings.SplitN(repoFull, "/", 2)
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "git-chat", parts[0]+"-"+parts[1])
}

func CloneOrPull(repoFull, localPath string) error {
	if _, err := os.Stat(filepath.Join(localPath, ".git")); os.IsNotExist(err) {
		cmd := exec.Command("gh", "repo", "clone", repoFull, localPath)
		_, err := cmd.CombinedOutput()
		return err
	}
	cmd := exec.Command("git", "-C", localPath, "pull", "--rebase", "origin", "main")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func EnsureCloned(repoFull, localPath string) error {
	if _, err := os.Stat(filepath.Join(localPath, ".git")); os.IsNotExist(err) {
		cmd := exec.Command("gh", "repo", "clone", repoFull, localPath)
		_, err := cmd.CombinedOutput()
		return err
	}
	return nil
}

func PullIfNew(repoFull, localPath string) (bool, error) {
	if err := EnsureCloned(repoFull, localPath); err != nil {
		return false, err
	}
	cmd := exec.Command("git", "-C", localPath, "fetch", "origin")
	if _, err := cmd.CombinedOutput(); err != nil {
		return false, nil
	}
	cmd = exec.Command("git", "-C", localPath, "rev-list", "--count", "HEAD..origin/main")
	count, _ := cmd.CombinedOutput()
	if strings.TrimSpace(string(count)) == "0" {
		return false, nil
	}
	cmd = exec.Command("git", "-C", localPath, "pull", "--rebase", "origin", "main")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git pull failed: %s", strings.TrimSpace(string(out)))
	}
	return true, nil
}
