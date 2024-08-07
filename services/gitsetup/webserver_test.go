package gitsetup

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	awsECR "github.com/aws/aws-sdk-go-v2/service/ecr"
	localECR "github.com/lep13/AutoBuildGo/services/ecr"
)

// Mock implementation of ECRClientInterface
func mockCreateECRClient() (*awsECR.Client, error) {
	return &awsECR.Client{}, nil
}

func mockCreateECRClientError() (*awsECR.Client, error) {
	return nil, errors.New("mock error creating ECR client")
}

func mockCreateRepo(repoName string, client localECR.ECRClientInterface) error {
	return nil
}

func mockCreateRepoError(repoName string, client localECR.ECRClientInterface) error {
	return errors.New("mock error creating ECR repository")
}

func mockCloneAndPushRepo(repoName string) error {
	return nil
}

func mockCloneAndPushRepoError(repoName string) error {
	return errors.New("mock error cloning and pushing repository")
}

func mockNewGitClient() *GitClient {
	return &GitClient{
		HTTPClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewBufferString("")),
				}, nil
			},
		},
		FetchSecretFunc: mockFetchSecretFunc,
	}
}

func mockNewGitClientError() *GitClient {
	return &GitClient{
		HTTPClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
				}, nil
			},
		},
		FetchSecretFunc: mockFetchSecretFunc,
	}
}

func mockDefaultRepoConfig(repoName, description string) (RepoConfig, error) {
	return RepoConfig{}, nil
}

func mockDefaultRepoConfigError(repoName, description string) (RepoConfig, error) {
	return RepoConfig{}, errors.New("mock error creating default repo config")
}

func TestCreateRepoHandler(t *testing.T) {
	// Mock the SleepFunc for the tests
	originalSleepFunc := SleepFunc
	SleepFunc = func(d time.Duration) {}
	defer func() { SleepFunc = originalSleepFunc }()

	tests := []struct {
		name           string
		body           RepoRequest
		createECRFunc  func() (*awsECR.Client, error)
		createRepoFunc func(string, localECR.ECRClientInterface) error
		newGitClient   func() *GitClient
		cloneAndPush   func(string) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Successful Repository Creation",
			body: RepoRequest{
				RepoName:    "test-repo",
				Description: "test description",
			},
			createECRFunc:  mockCreateECRClient,
			createRepoFunc: mockCreateRepo,
			newGitClient:   mockNewGitClient,
			cloneAndPush:   mockCloneAndPushRepo,
			expectedStatus: http.StatusOK,
			expectedBody:   "ECR and Git repositories created successfully",
		},
		{
			name:           "Invalid Method",
			body:           RepoRequest{},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Empty Repo Name",
			body:           RepoRequest{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Repository name is required",
		},
		{
			name: "Error Creating ECR Client",
			body: RepoRequest{
				RepoName:    "test-repo",
				Description: "test description",
			},
			createECRFunc:  mockCreateECRClientError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to create ECR client: mock error creating ECR client",
		},
		{
			name: "Error Creating ECR Repository",
			body: RepoRequest{
				RepoName:    "test-repo",
				Description: "test description",
			},
			createECRFunc:  mockCreateECRClient,
			createRepoFunc: mockCreateRepoError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to create ECR repository: mock error creating ECR repository",
		},
		// {
		// 	name: "Error Creating Git Repository",
		// 	body: RepoRequest{
		// 		RepoName:    "test-repo",
		// 		Description: "test description",
		// 	},
		// 	createECRFunc:  mockCreateECRClient,
		// 	createRepoFunc: mockCreateRepo,
		// 	newGitClient:   mockNewGitClientError,
		// 	expectedStatus: http.StatusInternalServerError,
		// 	expectedBody:   "Failed to create Git repository: Internal Server Error",
		// },
		{
			name: "Error Cloning and Pushing Repository",
			body: RepoRequest{
				RepoName:    "test-repo",
				Description: "test description",
			},
			createECRFunc:  mockCreateECRClient,
			createRepoFunc: mockCreateRepo,
			newGitClient:   mockNewGitClient,
			cloneAndPush:   mockCloneAndPushRepoError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Failed to clone and push repository: mock error cloning and pushing repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the mock functions
			CreateECRClientFunc = tt.createECRFunc
			CreateRepoFunc = tt.createRepoFunc
			NewGitClientFunc = tt.newGitClient
			CloneAndPushRepoFunc = tt.cloneAndPush

			// Create a request
			var req *http.Request
			if tt.name == "Invalid Method" {
				req = httptest.NewRequest(http.MethodGet, "/create-repo", nil)
			} else {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(http.MethodPost, "/create-repo", bytes.NewBuffer(body))
			}
			w := httptest.NewRecorder()

			// Call the handler
			CreateRepoHandler(w, req)

			// Check the response
			resp := w.Result()
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			bodyStr := strings.TrimSpace(string(body))
			expectedBodyStr := strings.TrimSpace(tt.expectedBody)
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
			if bodyStr != expectedBodyStr {
				t.Errorf("expected body %q, got %q", expectedBodyStr, bodyStr)
			}
		})
	}
}
func TestHandleWebServer(t *testing.T) {
	// Run the server in a goroutine
	go func() {
		HandleWebServer()
	}()

	// Wait a short time to ensure the server has started
	time.Sleep(100 * time.Millisecond)

	// Send a request to the server
	resp, err := http.Get("http://localhost:8082/create-repo")
	if err != nil {
		t.Fatalf("Failed to send request to server: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}
func TestCreateRepoHandler_BadRequest(t *testing.T) {
	// Test bad request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/create-repo", strings.NewReader("{invalid json}"))
	w := httptest.NewRecorder()

	CreateRepoHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestCreateRepoHandler_DefaultDescription(t *testing.T) {
	// Test default description when none is provided
	reqBody := RepoRequest{
		RepoName: "test-repo",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/create-repo", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	// Mock dependencies
	CreateECRClientFunc = mockCreateECRClient
	CreateRepoFunc = mockCreateRepo
	NewGitClientFunc = mockNewGitClient
	CloneAndPushRepoFunc = mockCloneAndPushRepo

	CreateRepoHandler(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// func TestCreateRepoHandler_ErrorCreatingDefaultRepoConfig(t *testing.T) {
// 	// Mock the DefaultRepoConfig function to simulate an error
// 	originalDefaultRepoConfig := DefaultRepoConfig
// 	DefaultRepoConfig = mockDefaultRepoConfigError
// 	defer func() { DefaultRepoConfig = originalDefaultRepoConfig }()

// 	reqBody := RepoRequest{
// 		RepoName:    "test-repo",
// 		Description: "test description",
// 	}
// 	body, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest(http.MethodPost, "/create-repo", bytes.NewBuffer(body))
// 	w := httptest.NewRecorder()

// 	// Mock dependencies
// 	CreateECRClientFunc = mockCreateECRClient
// 	CreateRepoFunc = mockCreateRepo
// 	NewGitClientFunc = mockNewGitClient
// 	CloneAndPushRepoFunc = mockCloneAndPushRepo

// 	CreateRepoHandler(w, req)

// 	resp := w.Result()
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusInternalServerError {
// 		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
// 	}
// 	if !strings.Contains(string(body), "Failed to create default repository configuration") {
// 		t.Errorf("expected error message not found, got %s", body)
// 	}
// }

// func TestCreateRepoHandler_ErrorCreatingGitRepository(t *testing.T) {
// 	// Mock the NewGitClient function to simulate an error in creating Git repository
// 	NewGitClientFunc = mockNewGitClientError

// 	reqBody := RepoRequest{
// 		RepoName:    "test-repo",
// 		Description: "test description",
// 	}
// 	body, _ := json.Marshal(reqBody)
// 	req := httptest.NewRequest(http.MethodPost, "/create-repo", bytes.NewBuffer(body))
// 	w := httptest.NewRecorder()

// 	// Mock dependencies
// 	CreateECRClientFunc = mockCreateECRClient
// 	CreateRepoFunc = mockCreateRepo
// 	CloneAndPushRepoFunc = mockCloneAndPushRepo

// 	CreateRepoHandler(w, req)

// 	resp := w.Result()
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusInternalServerError {
// 		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, resp.StatusCode)
// 	}
// 	if !strings.Contains(string(body), "Failed to create Git repository") {
// 		t.Errorf("expected error message not found, got %s", body)
// 	}
// }

// func TestHandleWebServer_ErrorStartingServer(t *testing.T) {
// 	// Mock the log.Fatalf function to capture the output
// 	originalLogFatalf := log.Fatalf
// 	defer func() { log.Fatalf = originalLogFatalf }()
// 	var logOutput bytes.Buffer
// 	log.SetOutput(&logOutput)
// 	log.Fatalf = func(format string, v ...interface{}) {
// 		logOutput.WriteString(fmt.Sprintf(format, v...))
// 	}

// 	// Mock the http.ListenAndServe function to simulate an error
// 	originalListenAndServe := http.ListenAndServe
// 	defer func() { http.ListenAndServe = originalListenAndServe }()
// 	http.ListenAndServe = func(addr string, handler http.Handler) error {
// 		return errors.New("mock error starting server")
// 	}

// 	HandleWebServer()

// 	if !strings.Contains(logOutput.String(), "Server failed to start") {
// 		t.Errorf("expected log output to contain 'Server failed to start', got %s", logOutput.String())
// 	}
// }
