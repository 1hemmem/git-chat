package auth

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2"
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

func RefreshAuthTerminal(scopes ...string) error {
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

func EnsureScope(scope string) error {
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
	if err := RefreshAuthTerminal(scope); err != nil {
		return err
	}
	fmt.Println("Auth refreshed successfully.")
	return nil
}
