package gitsetup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// GitHubService interface
type GitHubService interface {
	FetchSecretToken() (string, error)
	FetchGitHubUsername(token string) (string, error)
}

// DefaultGitHubService struct
type DefaultGitHubService struct{}

func (d DefaultGitHubService) FetchSecretToken() (string, error) {
	return FetchSecretToken() // Using the function defined in fetchsecrets.go
}

func (d DefaultGitHubService) FetchGitHubUsername(token string) (string, error) {
	return FetchGitHubUsername(token)
}

// Global variables to allow mocking in tests
var (
	gitHubService GitHubService = DefaultGitHubService{}
	execCommand                 = exec.Command
	readFile                    = os.ReadFile
	writeFile                   = os.WriteFile
	chdir                       = os.Chdir
	mkdirTemp                   = os.MkdirTemp
	removeAll                   = os.RemoveAll
)

// Define a variable to hold the HTTP client, which can be overridden in tests.
var httpClient = &http.Client{}

// CloneAndPushRepo clones the repository, updates the go.mod file, and pushes the changes back to GitHub.
func CloneAndPushRepo(repoName string) error {
	// Fetch GitHub token
	token, err := gitHubService.FetchSecretToken()
	if err != nil {
		return fmt.Errorf("error fetching GitHub token: %v", err)
	}

	// Fetch GitHub username
	username, err := gitHubService.FetchGitHubUsername(token)
	if err != nil {
		return fmt.Errorf("error fetching GitHub username: %v", err)
	}

	// Clone the repository
	repoURL := fmt.Sprintf("https://%s@github.com/%s/%s.git", token, username, repoName)
	cmd := execCommand("git", "clone", repoURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error cloning repository: %v", err)
	}

	// Change directory to the cloned repository
	if err := chdir(repoName); err != nil {
		return fmt.Errorf("error changing directory to cloned repository: %v", err)
	}

	// Update go.mod file
	goModFile := "go.mod"
	input, err := readFile(goModFile)
	if err != nil {
		return fmt.Errorf("error reading go.mod file: %v", err)
	}

	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "module") {
			lines[i] = fmt.Sprintf("module github.com/%s/%s", username, repoName)
			break
		}
	}
	output := strings.Join(lines, "\n")
	if err := writeFile(goModFile, []byte(output), 0644); err != nil {
		return fmt.Errorf("error writing to go.mod file: %v", err)
	}

	// Commit and push changes
	cmd = execCommand("git", "add", goModFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error adding go.mod file to git: %v", err)
	}

	cmd = execCommand("git", "commit", "-m", "Update go.mod module path")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error committing changes: %v", err)
	}

	cmd = execCommand("git", "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error pushing changes: %v", err)
	}

	// Go back to the previous directory
	if err := chdir(".."); err != nil {
		return fmt.Errorf("error changing back to the parent directory: %v", err)
	}

	// Remove the cloned repository
	if err := removeAll(repoName); err != nil {
		return fmt.Errorf("error removing the cloned repository: %v", err)
	}

	return nil
}

// FetchGitHubUsername fetches the GitHub username of the authenticated user.
func FetchGitHubUsername(token string, url ...string) (string, error) {
	requestURL := "https://api.github.com/user"
	if len(url) > 0 {
		requestURL = url[0]
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch GitHub username, status code: %d", resp.StatusCode)
	}

	var result struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Login, nil
}
