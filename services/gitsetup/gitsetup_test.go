package gitsetup

import (
	"bytes"
	"errors"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitClient{
				HTTPClient:      &mockHTTPClient{doFunc: tt.doFunc},
				FetchSecretFunc: tt.fetchSecretFunc,
			}

			err := client.CreateGitRepository(tt.config)
			if (err != nil) != (tt.expectedErrMessage != "") {
				t.Errorf("expected error: %v, got: %v", tt.expectedErrMessage != "", err)
			}
			if err != nil && err.Error() != tt.expectedErrMessage {
				t.Errorf("expected error message: %s, got: %s", tt.expectedErrMessage, err.Error())
			}
		})
	}
}
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
