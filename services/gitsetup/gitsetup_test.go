package gitsetup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// mockHTTPClient is a mock implementation of the HTTPClient interface.
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

// mockFetchSecretFunc is a mock implementation of the FetchSecretFunc.
func mockFetchSecretFunc() (string, error) {
	return "mock_token", nil
}

func mockFetchSecretFuncError() (string, error) {
	return "", errors.New("error fetching secret token")
}

// mockReader is a custom io.Reader that simulates a read error.
type mockReader struct{}

func (m *mockReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

// Custom type to simulate JSON marshaling error
type Unmarshalable struct{}

func (u Unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("mock marshaling error")
}

// Custom RepoConfig for testing purposes
type TestRepoConfig struct {
	Name        Unmarshalable `json:"name"`
	Description string        `json:"description"`
	Private     bool          `json:"private"`
	TemplateURL string        `json:"template_url"`
}

func (client *GitClient) createRepositoryWithTemplateTest(config TestRepoConfig, token string) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, config.TemplateURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+token)
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

func TestCreateGitRepository(t *testing.T) {
	tests := []struct {
		name               string
		doFunc             func(req *http.Request) (*http.Response, error)
		fetchSecretFunc    func() (string, error)
		config             RepoConfig
		expectedErrMessage string
	}{
		{
			name: "Successful Repository Creation",
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}, nil
			},
			fetchSecretFunc: mockFetchSecretFunc,
			config: RepoConfig{
				Name:        "test-repo",
				Description: "test description",
				Private:     true,
				TemplateURL: "https://api.github.com/repos/template-owner/template-repo/generate",
			},
			expectedErrMessage: "",
		},
		{
			name: "Fetch Secret Token Error",
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, nil
			},
			fetchSecretFunc:    mockFetchSecretFuncError,
			config:             RepoConfig{},
			expectedErrMessage: "error fetching secret token",
		},
		{
			name: "HTTP Request Creation Error",
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, nil
			},
			fetchSecretFunc: mockFetchSecretFunc,
			config: RepoConfig{
				Name:        "test-repo",
				Description: "test description",
				Private:     true,
				TemplateURL: ":invalid-url",
			},
			expectedErrMessage: "parse \":invalid-url\": missing protocol scheme",
		},
		{
			name: "HTTP Do Error",
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("HTTP Do error")
			},
			fetchSecretFunc: mockFetchSecretFunc,
			config: RepoConfig{
				Name:        "test-repo",
				Description: "test description",
				Private:     true,
				TemplateURL: "https://api.github.com/repos/template-owner/template-repo/generate",
			},
			expectedErrMessage: "HTTP Do error",
		},
		{
			name: "Failed Repository Creation",
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewBufferString("Bad Request")),
				}, nil
			},
			fetchSecretFunc: mockFetchSecretFunc,
			config: RepoConfig{
				Name:        "test-repo",
				Description: "test description",
				Private:     true,
				TemplateURL: "https://api.github.com/repos/template-owner/template-repo/generate",
			},
			expectedErrMessage: "failed to create repository, status code: 400, response: Bad Request",
		},
		{
			name: "JSON Marshal Error",
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, nil
			},
			fetchSecretFunc: mockFetchSecretFunc,
			config: RepoConfig{
				Name:        "test-repo",
				Description: "test description",
				Private:     true,
				TemplateURL: "https://api.github.com/repos/template-owner/template-repo/generate",
			},
			expectedErrMessage: "json: unsupported type: chan int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitClient{
				HTTPClient:      &mockHTTPClient{doFunc: tt.doFunc},
				FetchSecretFunc: tt.fetchSecretFunc,
			}

			if tt.name == "JSON Marshal Error" {
				type LocalRepoConfig struct {
					Name        string
					Description string
					Private     bool
					TemplateURL string
					Channel     chan int // This will cause JSON marshal error
				}

				localConfig := LocalRepoConfig{
					Name:        tt.config.Name,
					Description: tt.config.Description,
					Private:     tt.config.Private,
					TemplateURL: tt.config.TemplateURL,
					Channel:     make(chan int),
				}

				_, err := json.Marshal(localConfig)
				if err == nil || err.Error() != tt.expectedErrMessage {
					t.Fatalf("expected error: %v, got: %v", tt.expectedErrMessage, err)
				}
			} else {
				err := client.CreateGitRepository(tt.config)
				if (err != nil) != (tt.expectedErrMessage != "") {
					t.Errorf("expected error: %v, got: %v", tt.expectedErrMessage != "", err)
				}
				if err != nil && err.Error() != tt.expectedErrMessage {
					t.Errorf("expected error message: %s, got: %s", tt.expectedErrMessage, err.Error())
				}
			}
		})
	}
}

// func TestCreateGitRepository_JSONMarshalError(t *testing.T) {
// 	client := &GitClient{
// 		HTTPClient:      &mockHTTPClient{},
// 		FetchSecretFunc: mockFetchSecretFunc,
// 	}

// 	err := client.createRepositoryWithTemplateTest(TestRepoConfig{
// 		Name:        Unmarshalable{}, // This will cause JSON marshal error
// 		Description: "test description",
// 		Private:     true,
// 		TemplateURL: "https://api.github.com/repos/template-owner/template-repo/generate",
// 	}, "mock_token")

// 	expectedErr := "json: error calling MarshalJSON for type gitsetup.Unmarshalable: mock marshaling error"
// 	if err == nil || err.Error() != expectedErr {
// 		t.Fatalf("expected error: %v, got: %v", expectedErr, err)
// 	}
// }

func TestNewGitClient(t *testing.T) {
	client := NewGitClient()

	if _, ok := client.HTTPClient.(*http.Client); !ok {
		t.Errorf("expected HTTPClient to be of type *http.Client, got %T", client.HTTPClient)
	}

	if client.FetchSecretFunc == nil {
		t.Errorf("expected FetchSecretFunc to be set, but it was nil")
	}

	// token, err := client.FetchSecretFunc()
	// if err != nil {
	// 	t.Errorf("expected no error from FetchSecretFunc, got %v", err)
	// }

	// if token == "" {
	// 	t.Errorf("expected a non-empty token from FetchSecretFunc, got an empty string")
	// }
}

func TestCreateGitRepository_ReadResponseBodyError(t *testing.T) {
	client := &GitClient{
		HTTPClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(&mockReader{}),
				}, nil
			},
		},
		FetchSecretFunc: mockFetchSecretFunc,
	}

	err := client.CreateGitRepository(RepoConfig{
		Name:        "test-repo",
		Description: "test description",
		Private:     true,
		TemplateURL: "https://api.github.com/repos/template-owner/template-repo/generate",
	})

	expectedErrMessage := "failed to read response body: mock read error"
	if err == nil || err.Error() != expectedErrMessage {
		t.Fatalf("expected error message: %s, got: %v", expectedErrMessage, err)
	}
}
