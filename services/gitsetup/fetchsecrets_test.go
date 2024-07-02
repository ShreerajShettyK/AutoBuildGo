package gitsetup

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type mockConfigLoader struct {
	err error
}

func (m *mockConfigLoader) LoadDefaultConfig(ctx context.Context, options ...func(*config.LoadOptions) error) (aws.Config, error) {
	if m.err != nil {
		return aws.Config{}, m.err
	}
	return aws.Config{}, nil
}

type mockSecretsManagerClient struct {
	secretString string
	err          error
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(m.secretString),
	}, nil
}

// MockLogger is a mock implementation of the io.Writer interface to capture log.Fatalf calls
type MockLogger struct {
	FatalMsg string
}

func (m *MockLogger) Write(p []byte) (n int, err error) {
	m.FatalMsg = string(p)
	return len(p), nil
}

func TestFetchSecretValue(t *testing.T) {
	secretData := map[string]string{
		"GITHUB_TOKEN": "test_github_token",
		"TEMPLATE_URL": "test_template_url",
	}
	secretString, _ := json.Marshal(secretData)

	tests := []struct {
		name          string
		secretString  string
		err           error
		key           string
		expectedValue string
		expectedErr   bool
	}{
		{
			name:          "Successful Fetch",
			secretString:  string(secretString),
			key:           "GITHUB_TOKEN",
			expectedValue: "test_github_token",
			expectedErr:   false,
		},
		{
			name:         "Key Not Found",
			secretString: string(secretString),
			key:          "INVALID_KEY",
			expectedErr:  true,
		},
		{
			name:        "Error Fetching Secret",
			err:         errors.New("error fetching secret"),
			key:         "GITHUB_TOKEN",
			expectedErr: true,
		},
		{
			name:         "Error Unmarshalling Secret",
			secretString: `invalid_json`,
			key:          "GITHUB_TOKEN",
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configLoader = &mockConfigLoader{}
			secretsManagerClient = &mockSecretsManagerClient{
				secretString: tt.secretString,
				err:          tt.err,
			}

			// Clear the cache before each test
			secretCache.Lock()
			secretCache.data = make(map[string]string)
			secretCache.Unlock()

			value, err := FetchSecretValue(tt.key)
			if (err != nil) != tt.expectedErr {
				t.Errorf("expected error: %v, got: %v", tt.expectedErr, err)
			}
			if value != tt.expectedValue {
				t.Errorf("expected value: %s, got: %s", tt.expectedValue, value)
			}
		})
	}
}

func TestFetchSecretToken(t *testing.T) {
	secretData := map[string]string{
		"GITHUB_TOKEN": "test_github_token",
	}
	secretString, _ := json.Marshal(secretData)

	configLoader = &mockConfigLoader{}
	secretsManagerClient = &mockSecretsManagerClient{
		secretString: string(secretString),
		err:          nil,
	}

	// Clear the cache before the test
	secretCache.Lock()
	secretCache.data = make(map[string]string)
	secretCache.Unlock()

	token, err := FetchSecretToken()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if token != "test_github_token" {
		t.Errorf("expected token: %s, got: %s", "test_github_token", token)
	}
}

func TestFetchTemplateURL(t *testing.T) {
	secretData := map[string]string{
		"TEMPLATE_URL": "test_template_url",
	}
	secretString, _ := json.Marshal(secretData)

	configLoader = &mockConfigLoader{}
	secretsManagerClient = &mockSecretsManagerClient{
		secretString: string(secretString),
		err:          nil,
	}

	// Clear the cache before the test
	secretCache.Lock()
	secretCache.data = make(map[string]string)
	secretCache.Unlock()

	url, err := FetchTemplateURL()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if url != "test_template_url" {
		t.Errorf("expected URL: %s, got: %s", "test_template_url", url)
	}
}

func TestFetchSecretValue_ConfigLoaderError(t *testing.T) {
	originalConfigLoader := configLoader
	defer func() { configLoader = originalConfigLoader }()

	configLoader = &mockConfigLoader{
		err: errors.New("mock error loading config"),
	}

	value, err := FetchSecretValue("GITHUB_TOKEN")
	if err == nil || !strings.Contains(err.Error(), "error loading AWS config") {
		t.Errorf("expected error loading AWS config, got %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value, got %s", value)
	}
}

func TestDefaultCommandRunner_Run(t *testing.T) {
	runner := &DefaultCommandRunner{}
	cmd := exec.Command("cmd", "/c", "echo", "test")
	err := runner.Run(cmd)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDefaultCommandRunner_Output(t *testing.T) {
	runner := &DefaultCommandRunner{}
	cmd := exec.Command("cmd", "/c", "echo", "test")
	output, err := runner.Output(cmd)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	expectedOutput := "test\r\n"
	if string(output) != expectedOutput {
		t.Errorf("expected output: %s, got: %s", expectedOutput, string(output))
	}
}

// func TestInit_LogFatalf(t *testing.T) {
// 	originalLogger := log.Default()
// 	mockLogger := &MockLogger{}
// 	log.SetOutput(mockLogger)
// 	defer log.SetOutput(os.Stderr)

// 	// Simulate configuration loading error
// 	originalConfigLoader := configLoader
// 	configLoader = &mockConfigLoader{
// 		err: errors.New("mock error loading config"),
// 	}
// 	defer func() { configLoader = originalConfigLoader }()

// 	func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				// Catch the log.Fatalf panic
// 				if mockLogger.FatalMsg == "" {
// 					t.Errorf("expected log.Fatalf to be called")
// 				}
// 			}
// 		}()
// 		init() // Call the init function directly
// 	}()

// 	if mockLogger.FatalMsg == "" {
// 		t.Errorf("expected log.Fatalf to be called")
// 	}
// }