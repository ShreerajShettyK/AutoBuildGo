package gitsetup

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// MockGitHubService struct
type MockGitHubService struct {
	MockFetchSecretToken    func() (string, error)
	MockFetchGitHubUsername func(token string) (string, error)
}

func (m *MockGitHubService) FetchSecretToken() (string, error) {
	return m.MockFetchSecretToken()
}

func (m *MockGitHubService) FetchGitHubUsername(token string) (string, error) {
	return m.MockFetchGitHubUsername(token)
}

// Mock exec.Command function
func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// Mock exec.Command function for simulating errors
func mockExecCommandError(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "GO_COMMAND_ERROR=1"}
	return cmd
}

// Extend the TestHelperProcess function to simulate errors
func TestHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if os.Getenv("GO_COMMAND_ERROR") == "1" {
		fmt.Fprintln(os.Stderr, "mocked command execution error")
		os.Exit(1)
	}
	fmt.Println("mocked command execution")
	os.Exit(0)
}

// Mock file operations
var (
	mockReadFile = func(filename string) ([]byte, error) {
		if filename == "go.mod" {
			return []byte("module example.com/testrepo\n"), nil
		}
		return nil, fmt.Errorf("file not found")
	}
	mockWriteFileContent string
	mockWriteFile        = func(filename string, data []byte, perm os.FileMode) error {
		if filename == "go.mod" {
			mockWriteFileContent = string(data)
			return nil
		}
		return fmt.Errorf("cannot write to file")
	}
	mockChdir = func(dir string) error {
		if dir == "testrepo" || dir == ".." {
			return nil
		}
		return fmt.Errorf("chdir: %s: no such file or directory", dir)
	}
	mockMkdirTemp = func(dir, prefix string) (string, error) {
		return "mocked_temp_dir", nil
	}
	mockRemoveAll = func(path string) error {
		return nil
	}
)

func TestCloneAndPushRepo(t *testing.T) {
	// Override the global variables within this test
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	originalMkdirTemp := mkdirTemp
	originalRemoveAll := removeAll

	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
		mkdirTemp = originalMkdirTemp
		removeAll = originalRemoveAll
	}()

	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommand
	readFile = mockReadFile
	writeFile = mockWriteFile
	chdir = mockChdir
	mkdirTemp = mockMkdirTemp
	removeAll = mockRemoveAll

	// Redirect stdout and stderr to avoid actual output
	var outBuf, errBuf bytes.Buffer
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()
	stdout := io.MultiWriter(&outBuf, originalStdout)
	stderr := io.MultiWriter(&errBuf, originalStderr)

	stdoutReader, stdoutWriter, _ := os.Pipe()
	stderrReader, stderrWriter, _ := os.Pipe()

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	go func() {
		io.Copy(stdout, stdoutReader)
	}()
	go func() {
		io.Copy(stderr, stderrReader)
	}()

	err := CloneAndPushRepo("testrepo")
	if err != nil {
		t.Fatalf("CloneAndPushRepo failed: %v", err)
	}

	expectedContent := "module github.com/mocked_user/testrepo\n"
	if mockWriteFileContent != expectedContent {
		t.Errorf("unexpected go.mod content: %v", mockWriteFileContent)
	}
}

func TestFetchGitHubUsername(t *testing.T) {
	// Mocking the HTTP request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token mocked_token" {
			t.Fatalf("unexpected token: %v", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"login": "mocked_user"}`)
	}))
	defer ts.Close()

	originalHTTPClient := httpClient
	httpClient = ts.Client()
	defer func() { httpClient = originalHTTPClient }()

	originalGitHubService := gitHubService
	defer func() { gitHubService = originalGitHubService }()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
	}

	username, err := FetchGitHubUsername("mocked_token", ts.URL)
	if err != nil {
		t.Fatalf("FetchGitHubUsername failed: %v", err)
	}

	expectedUsername := "mocked_user"
	if username != expectedUsername {
		t.Errorf("unexpected username: got %v, want %v", username, expectedUsername)
	}
}
func TestCloneAndPushRepo_ErrorFetchingToken(t *testing.T) {
	originalGitHubService := gitHubService
	defer func() { gitHubService = originalGitHubService }()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "", fmt.Errorf("mocked error fetching token")
		},
	}

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error fetching GitHub token: mocked error fetching token" {
		t.Fatalf("expected error fetching token, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorFetchingUsername(t *testing.T) {
	originalGitHubService := gitHubService
	defer func() { gitHubService = originalGitHubService }()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "", fmt.Errorf("mocked error fetching username")
		},
	}

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error fetching GitHub username: mocked error fetching username" {
		t.Fatalf("expected error fetching username, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorChangingDirectory(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommand
	chdir = func(dir string) error {
		if dir == "testrepo" {
			return fmt.Errorf("mocked error changing directory")
		}
		return nil
	}

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error changing directory to cloned repository: mocked error changing directory" {
		t.Fatalf("expected error changing directory, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorReadingGoModFile(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommand
	chdir = mockChdir
	readFile = func(filename string) ([]byte, error) {
		return nil, fmt.Errorf("mocked error reading go.mod file")
	}

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error reading go.mod file: mocked error reading go.mod file" {
		t.Fatalf("expected error reading go.mod file, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorWritingGoModFile(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommand
	chdir = mockChdir
	readFile = mockReadFile
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		return fmt.Errorf("mocked error writing to go.mod file")
	}

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error writing to go.mod file: mocked error writing to go.mod file" {
		t.Fatalf("expected error writing to go.mod file, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorChangingBackToParentDirectory(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommand
	chdir = func(dir string) error {
		if dir == ".." {
			return fmt.Errorf("mocked error changing back to parent directory")
		}
		return nil
	}
	readFile = mockReadFile
	writeFile = mockWriteFile

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error changing back to the parent directory: mocked error changing back to parent directory" {
		t.Fatalf("expected error changing back to parent directory, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorRemovingClonedRepository(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	originalRemoveAll := removeAll
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
		removeAll = originalRemoveAll
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommand
	chdir = mockChdir
	readFile = mockReadFile
	writeFile = mockWriteFile
	removeAll = func(path string) error {
		return fmt.Errorf("mocked error removing the cloned repository")
	}

	err := CloneAndPushRepo("testrepo")
	if err == nil || err.Error() != "error removing the cloned repository: mocked error removing the cloned repository" {
		t.Fatalf("expected error removing the cloned repository, got: %v", err)
	}
}

func TestFetchGitHubUsername_ErrorStatusCode(t *testing.T) {
	// Mocking the HTTP request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, `{"message": "Bad credentials"}`)
	}))
	defer ts.Close()

	originalHTTPClient := httpClient
	httpClient = ts.Client()
	defer func() { httpClient = originalHTTPClient }()

	originalGitHubService := gitHubService
	defer func() { gitHubService = originalGitHubService }()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
	}

	_, err := FetchGitHubUsername("mocked_token", ts.URL)
	if err == nil || !strings.Contains(err.Error(), "failed to fetch GitHub username, status code: 401") {
		t.Fatalf("expected error fetching GitHub username, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorCloningRepository(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = mockExecCommandError

	err := CloneAndPushRepo("testrepo")
	if err == nil || !strings.Contains(err.Error(), "error cloning repository") {
		t.Fatalf("expected error cloning repository, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorAddingGoModFileToGit(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name == "git" && arg[0] == "add" {
			return mockExecCommandError(name, arg...)
		}
		return mockExecCommand(name, arg...)
	}
	readFile = mockReadFile
	writeFile = mockWriteFile
	chdir = mockChdir

	err := CloneAndPushRepo("testrepo")
	if err == nil || !strings.Contains(err.Error(), "error adding go.mod file to git") {
		t.Fatalf("expected error adding go.mod file to git, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorCommittingChanges(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name == "git" && arg[0] == "commit" {
			return mockExecCommandError(name, arg...)
		}
		return mockExecCommand(name, arg...)
	}
	readFile = mockReadFile
	writeFile = mockWriteFile
	chdir = mockChdir

	err := CloneAndPushRepo("testrepo")
	if err == nil || !strings.Contains(err.Error(), "error committing changes") {
		t.Fatalf("expected error committing changes, got: %v", err)
	}
}

func TestCloneAndPushRepo_ErrorPushingChanges(t *testing.T) {
	originalGitHubService := gitHubService
	originalExecCommand := execCommand
	originalReadFile := readFile
	originalWriteFile := writeFile
	originalChdir := chdir
	defer func() {
		gitHubService = originalGitHubService
		execCommand = originalExecCommand
		readFile = originalReadFile
		writeFile = originalWriteFile
		chdir = originalChdir
	}()
	gitHubService = &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return "mocked_user", nil
		},
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if name == "git" && arg[0] == "push" {
			return mockExecCommandError(name, arg...)
		}
		return mockExecCommand(name, arg...)
	}
	readFile = mockReadFile
	writeFile = mockWriteFile
	chdir = mockChdir

	err := CloneAndPushRepo("testrepo")
	if err == nil || !strings.Contains(err.Error(), "error pushing changes") {
		t.Fatalf("expected error pushing changes, got: %v", err)
	}
}

// func TestDefaultGitHubService_FetchSecretToken(t *testing.T) {
// 	// Mock the FetchSecretToken to return a mock token
// 	mockService := &MockGitHubService{
// 		MockFetchSecretToken: func() (string, error) {
// 			return "mocked_token_value_for_testing", nil
// 		},
// 	}

// 	originalGitHubService := gitHubService
// 	gitHubService = mockService
// 	defer func() { gitHubService = originalGitHubService }()

// 	d := DefaultGitHubService{}
// 	token, err := d.FetchSecretToken()

// 	expectedToken := "mocked_token_value_for_testing"
// 	if err != nil || token != expectedToken {
// 		t.Fatalf("expected token to be '%s', got: '%v', err: %v", expectedToken, token, err)
// 	}
// }

func TestDefaultGitHubService_FetchGitHubUsername_Error(t *testing.T) {
	// Mocking the HTTP request to simulate an error response
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token mocked_token" {
			t.Fatalf("unexpected token: %v", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, `{"message": "Bad credentials"}`)
	}))
	defer ts.Close()

	// Replace the httpClient with the test server client
	originalHTTPClient := httpClient
	httpClient = ts.Client()
	defer func() { httpClient = originalHTTPClient }()

	// Use the existing MockGitHubService struct
	mockService := &MockGitHubService{
		MockFetchSecretToken: func() (string, error) {
			return "mocked_token", nil
		},
		MockFetchGitHubUsername: func(token string) (string, error) {
			return FetchGitHubUsername(token, ts.URL)
		},
	}

	// Test the DefaultGitHubService.FetchGitHubUsername method
	originalGitHubService := gitHubService
	gitHubService = mockService
	defer func() { gitHubService = originalGitHubService }()

	service := DefaultGitHubService{}
	username, err := service.FetchGitHubUsername("mocked_token")
	if err == nil || username != "" {
		t.Fatalf("expected error fetching GitHub username, got: '%v', username: '%v'", err, username)
	}
	expectedError := "failed to fetch GitHub username, status code: 401"
	if err.Error() != expectedError {
		t.Errorf("unexpected error: got %v, want %v", err, expectedError)
	}
}
