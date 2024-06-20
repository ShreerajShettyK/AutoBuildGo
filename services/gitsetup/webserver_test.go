package gitsetup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"os/exec"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/google/uuid"
	myecr "github.com/lep13/AutoBuildGo/services/ecr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockECRClient mocks the ECRClientInterface from the ecr package.
type MockECRClient struct {
	mock.Mock
}

// Implement the CreateRepository method of the ECRClientInterface
func (m *MockECRClient) CreateRepository(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
	args := m.Called(ctx, params, optFns)
	output, ok := args.Get(0).(*ecr.CreateRepositoryOutput)
	if !ok {
		output = nil
	}
	return output, args.Error(1)
}

// MockGitClient mocks the GitClient struct's CreateGitRepository method.
type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) CreateGitRepository(config RepoConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

// Define the mock function for CloneAndPushRepo specific to this test file.
var mockCloneAndPushRepo = func(repoName string) error {
	return nil
}

// MockCommandRunner implements CommandRunner interface for testing
type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(cmd *exec.Cmd) error {
	args := m.Called(cmd)
	return args.Error(0)
}

func (m *MockCommandRunner) Output(cmd *exec.Cmd) ([]byte, error) {
	args := m.Called(cmd)
	return args.Get(0).([]byte), args.Error(1)
}

// TestMain setup
func TestMain(m *testing.M) {
	// Set the TEMPLATE_URL environment variable for all tests
	os.Setenv("TEMPLATE_URL", "https://api.github.com/repos/lep13/ServiceTemplate/generate")
	defer os.Unsetenv("TEMPLATE_URL")

	// Override CloneAndPushRepoFunc with the mock version
	CloneAndPushRepoFunc = func(repoName string) error {
		return mockCloneAndPushRepo(repoName)
	}

	// Run the tests
	code := m.Run()

	// Exit with the test run code
	os.Exit(code)
}

// GenerateUniqueRepoName creates a truly unique repository name using UUID.
func GenerateUniqueRepoName(baseName string) string {
	return baseName + "-" + uuid.New().String()
}

// Mock sleep to bypass delays
var sleepFunc = func(d time.Duration) {
	time.Sleep(d)
}

// Test function for CreateRepoHandler
func TestCreateRepoHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		body               interface{}
		setupMocks         func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner)
		expectedStatusCode int
		expectedResponse   string
	}{
		// {
		// 	name:   "Valid request",
		// 	method: http.MethodPost,
		// 	body: RepoRequest{
		// 		RepoName:    GenerateUniqueRepoName("test-repo"), // Use a UUID based unique name
		// 		Description: "A test repository",
		// 	},
		// 	setupMocks: func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {
		// 		ecrClient.On("CreateRepository", mock.Anything, mock.Anything, mock.Anything).Return(&ecr.CreateRepositoryOutput{}, nil).Once()
		// 		gitClient.On("CreateGitRepository", mock.Anything).Return(nil).Once()

		// 		cmdRunner.On("Run", mock.Anything).Return(nil).Times(3) // Assuming three main steps: clone, commit, push
		// 		cmdRunner.On("Output", mock.Anything).Return([]byte(""), nil).Times(1) // Assuming one call to get command output

		// 		mockCloneAndPushRepo = func(repoName string) error {
		// 			return nil
		// 		}
		// 	},
		// 	expectedStatusCode: http.StatusOK,
		// 	expectedResponse:   "ECR and Git repositories created successfully",
		// },
		{
			name:               "Method not allowed",
			method:             http.MethodGet,
			body:               nil,
			setupMocks:         func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {},
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedResponse:   "Method not allowed\n",
		},
		{
			name:               "Bad request - invalid JSON",
			method:             http.MethodPost,
			body:               "invalid json",
			setupMocks:         func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse:   "Bad request\n",
		},
		{
			name:   "Bad request - missing repo name",
			method: http.MethodPost,
			body: RepoRequest{
				Description: "A test repository",
			},
			setupMocks:         func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse:   "Repository name is required\n",
		},
		{
			name:   "ECR repo creation failure",
			method: http.MethodPost,
			body: RepoRequest{
				RepoName:    GenerateUniqueRepoName("test-repo"), // Use a UUID based unique name
				Description: "A test repository",
			},
			setupMocks: func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {
				ecrClient.On("CreateRepository", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("ecr repo creation failed")).Once()
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse:   "Failed to create ECR repository: ecr repo creation failed\n",
		},
		// {
		// 	name:   "Git repo creation failure",
		// 	method: http.MethodPost,
		// 	body: RepoRequest{
		// 		RepoName:    GenerateUniqueRepoName("test-repo"), // Use a UUID based unique name
		// 		Description: "A test repository",
		// 	},
		// 	setupMocks: func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {
		// 		ecrClient.On("CreateRepository", mock.Anything, mock.Anything, mock.Anything).Return(&ecr.CreateRepositoryOutput{}, nil).Once()
		// 		gitClient.On("CreateGitRepository", mock.Anything).Return(errors.New("git repo creation failed")).Once()
		// 	},
		// 	expectedStatusCode: http.StatusInternalServerError,
		// 	expectedResponse:   "Failed to create Git repository: git repo creation failed\n",
		// },
		// {
		// 	name:   "Clone and push failure",
		// 	method: http.MethodPost,
		// 	body: RepoRequest{
		// 		RepoName:    GenerateUniqueRepoName("test-repo"), // Use a UUID based unique name
		// 		Description: "A test repository",
		// 	},
		// 	setupMocks: func(ecrClient *MockECRClient, gitClient *MockGitClient, cmdRunner *MockCommandRunner) {
		// 		ecrClient.On("CreateRepository", mock.Anything, mock.Anything, mock.Anything).Return(&ecr.CreateRepositoryOutput{}, nil).Once()
		// 		gitClient.On("CreateGitRepository", mock.Anything).Return(nil).Once()
		// 		mockCloneAndPushRepo = func(repoName string) error {
		// 			return errors.New("clone and push failed")
		// 		}
		// 	},
		// 	expectedStatusCode: http.StatusInternalServerError,
		// 	expectedResponse:   "Failed to clone and push repository: clone and push failed\n",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks for ECR, Git clients, and Command Runner
			mockECRClient := new(MockECRClient)
			mockGitClient := new(MockGitClient)
			mockCommandRunner := new(MockCommandRunner)

			// Set up mocks based on the test case
			if tt.setupMocks != nil {
				tt.setupMocks(mockECRClient, mockGitClient, mockCommandRunner)
			}

			// Prepare the request body
			var reqBody []byte
			if tt.body != nil {
				reqBody, _ = json.Marshal(tt.body)
			}

			// Create a new HTTP request
			req := httptest.NewRequest(tt.method, "/create-repo", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()

			// Override external dependencies for testing
			CreateECRClientFunc = func() (*ecr.Client, error) {
				return &ecr.Client{}, nil
			}
			CreateRepoFunc = func(repoName string, client myecr.ECRClientInterface) error {
				// Use the mock method for CreateRepository
				input := &ecr.CreateRepositoryInput{
					RepositoryName: &repoName,
				}
				_, err := mockECRClient.CreateRepository(context.Background(), input)
				return err
			}
			NewGitClientFunc = func() *GitClient {
				return &GitClient{
					HTTPClient:      &http.Client{},
					FetchSecretFunc: FetchSecretToken,
				}
			}
			LoadEnvFunc = func() {
				// Mock environment loading, do nothing
			}

			// Override command runner to use the mock
			runner = mockCommandRunner

			// Temporarily override the sleep function for testing to avoid delays
			originalSleepFunc := sleepFunc
			sleepFunc = func(d time.Duration) {
				// No-op for testing
			}
			defer func() { sleepFunc = originalSleepFunc }()

			// Invoke the handler with the request and response recorder
			CreateRepoHandler(rr, req)

			// Check the status code and response body
			assert.Equal(t, tt.expectedStatusCode, rr.Code)
			assert.Equal(t, tt.expectedResponse, rr.Body.String())

			// Verify all expectations
			mockECRClient.AssertExpectations(t)
			mockGitClient.AssertExpectations(t)
			mockCommandRunner.AssertExpectations(t)
		})
	}
}

// Integration test for the web server
func TestHandleWebServer(t *testing.T) {
	go func() {
		// Start the server in a goroutine
		HandleWebServer()
	}()

	// Give the server some time to start up
	time.Sleep(1 * time.Second)

	// Make a request to the server
	resp, err := http.Post("http://localhost:8080/create-repo", "application/json", bytes.NewBuffer([]byte(`{"RepoName":"integration-test-repo", "Description":"Integration test repo"}`)))
	if err != nil {
		t.Fatalf("Failed to make request to server: %v", err)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Validate the response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "ECR and Git repositories created successfully")

	// Further validation or cleanup can be added here
}
