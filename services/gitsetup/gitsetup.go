package gitsetup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// HTTPClient is an interface that defines the Do method used by http.Client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// GitClient is a structure that holds dependencies for making HTTP requests.
type GitClient struct {
	HTTPClient      HTTPClient
	FetchSecretFunc func() (string, error)
}

// NewGitClient returns an instance of GitClient with default dependencies.
func NewGitClient() *GitClient {
	return &GitClient{
		HTTPClient:      &http.Client{},
		FetchSecretFunc: FetchSecretToken, // Fetching token from Secrets Manager
	}
}

// Extract the template owner and repo from the TEMPLATE_URL
func parseTemplateURL(templateURL string) (string, string, error) {
	parsedURL, err := url.Parse(templateURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid TEMPLATE_URL: %v", err)
	}

	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 3 {
		return "", "", fmt.Errorf("invalid TEMPLATE_URL: %s", templateURL)
	}

	owner := pathParts[1]
	repo := pathParts[2]
	return owner, repo, nil
}

// CreateGitRepository creates a new GitHub repository using the specified configuration.
func (client *GitClient) CreateGitRepository(config RepoConfig, userToken string) error {
	secretToken, err := client.FetchSecretFunc() // Fetch the token for template access
	if err != nil {
		return err
	}

	// Extract template owner and repo from the TEMPLATE_URL
	templateURL := os.Getenv("TEMPLATE_URL")
	templateOwner, templateRepo, err := parseTemplateURL(templateURL)
	if err != nil {
		return err
	}

	return client.createRepositoryWithTemplate(config, secretToken, templateOwner, templateRepo, userToken)
}

// createRepositoryWithTemplate sends a request to GitHub API to create a repository from a template.
func (client *GitClient) createRepositoryWithTemplate(config RepoConfig, secretToken, templateOwner, templateRepo, userToken string) error {
	// Payload for creating a repository from a template
	data, err := json.Marshal(map[string]interface{}{
		"name":        config.Name,
		"description": config.Description,
		"private":     config.Private,
	})

	if err != nil {
		return err
	}

	// Construct the API endpoint for creating a repository from a template
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/generate", templateOwner, templateRepo)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	// Use the GitHub token for accessing the template
	req.Header.Set("Authorization", "token "+userToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	return fmt.Errorf("failed to create repository, status code: %d, response: %s", resp.StatusCode, string(body))
}

// Additional function to fetch the owner details from the token
func (client *GitClient) FetchOwnerDetails(token string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := client.HTTPClient.Do(req)
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
