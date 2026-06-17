package chat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"git-chat/internal/repo"
)

type Message struct {
	Author    string
	Timestamp string
	Body      string
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

func SendMessage(repoName, body string) error {
	repoFull, err := repo.ResolveGroup(repoName)
	if err != nil {
		return err
	}
	username, err := repo.GetGitHubUsername()
	if err != nil {
		return err
	}
	localPath := repo.CachePath(repoFull)
	if err := repo.EnsureCloned(repoFull, localPath); err != nil {
		return fmt.Errorf("failed to clone repository: %v", err)
	}
	msgsDir := filepath.Join(localPath, "messages")
	if err := os.MkdirAll(msgsDir, 0755); err != nil {
		return fmt.Errorf("failed to create messages directory: %v", err)
	}
	filename := generateFilename(username)
	filePath := filepath.Join(msgsDir, filename)
	if err := os.WriteFile(filePath, []byte(body), 0644); err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}
	cmd := exec.Command("git", "-C", localPath, "add", "messages/"+filename)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("staging message failed")
	}
	cmd = exec.Command("git", "-C", localPath, "commit", "-m", fmt.Sprintf("message from %s", username))
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("committing message failed")
	}
	cmd = exec.Command("git", "-C", localPath, "pull", "--rebase", "origin", "main")
	cmd.Run()
	cmd = exec.Command("git", "-C", localPath, "push", "origin", "main")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if strings.Contains(msg, "Permission denied") || strings.Contains(msg, "403") {
			return fmt.Errorf("push denied — you don't have write access to this group")
		}
		return fmt.Errorf("pushing message failed")
	}
	return nil
}

func ReadMessages(repoName string) ([]Message, error) {
	repoFull, err := repo.ResolveGroup(repoName)
	if err != nil {
		return nil, err
	}
	localPath := repo.CachePath(repoFull)
	if err := repo.CloneOrPull(repoFull, localPath); err != nil {
		return nil, fmt.Errorf("failed to clone/pull repository: %v", err)
	}
	return readFromDir(filepath.Join(localPath, "messages"))
}

func ReadMessagesFromCache(localPath string) ([]Message, error) {
	return readFromDir(filepath.Join(localPath, "messages"))
}

func readFromDir(msgsDir string) ([]Message, error) {
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
