package gitsetup

import (
	"fmt"
)

type RepoConfig struct {
	Name        string
	Description string
	Private     bool
	AutoInit    bool
	TemplateURL string
}

func DefaultRepoConfig(repoName string, description string) (RepoConfig, error) {
	templateURL, err := FetchTemplateURL()
	if err != nil {
		return RepoConfig{}, fmt.Errorf("failed to fetch template URL: %v", err)
	}

	return RepoConfig{
		Name:        repoName,
		Description: description,
		Private:     true,
		AutoInit:    true,
		TemplateURL: templateURL,
	}, nil
}
