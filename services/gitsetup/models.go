package gitsetup

type RepoConfig struct {
	Name        string
	Description string
	Private     bool
	AutoInit    bool
	TemplateURL string
}

type SecretData struct {
	GITHUB_TOKEN   string `json:"GITHUB_TOKEN"`
	GITHUB_USER    string `json:"GITHUB_USER"`
	GITHUB_PASSWORD string `json:"GITHUB_PASSWORD"`
}
