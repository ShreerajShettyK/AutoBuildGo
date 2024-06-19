package gitsetup

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConfigLoader mocks the ConfigLoader interface
type MockConfigLoader struct {
	mock.Mock
}

func (m *MockConfigLoader) LoadDefaultConfig(ctx context.Context, options ...func(*config.LoadOptions) error) (aws.Config, error) {
	args := m.Called(ctx, options)
	// Return the config and error based on mock setup
	if config, ok := args.Get(0).(aws.Config); ok {
		return config, args.Error(1)
	}
	return aws.Config{}, args.Error(1)
}

// MockSecretsManagerClient mocks the SecretsManagerClient interface
type MockSecretsManagerClient struct {
	mock.Mock
}

func (m *MockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, params, optFns)
	// Return the secret value and error based on mock setup
	if result, ok := args.Get(0).(*secretsmanager.GetSecretValueOutput); ok {
		return result, args.Error(1)
	}
	return nil, args.Error(1)
}

func resetGlobalState() {
	// Clear the cache and reset mocks to default states
	secretCache.Lock()
	secretCache.data = make(map[string]string)
	secretCache.Unlock()
}

func TestFetchSecretToken(t *testing.T) {
	// Backup the real instances
	realConfigLoader := configLoader
	realSecretsManagerClient := secretsManagerClient

	// Restore the real instances after the test completes
	defer func() {
		configLoader = realConfigLoader
		secretsManagerClient = realSecretsManagerClient
		resetGlobalState()
	}()

	tests := []struct {
		name          string
		mockConfig    func(*MockConfigLoader)
		mockSecret    func(*MockSecretsManagerClient)
		expectedToken string
		expectedError string
	}{
		{
			name: "success",
			mockConfig: func(m *MockConfigLoader) {
				m.On("LoadDefaultConfig", mock.Anything, mock.Anything).Return(aws.Config{Region: "us-east-1"}, nil).Once()
			},
			mockSecret: func(m *MockSecretsManagerClient) {
				secretString := `{"GITHUB_TOKEN": "test_token"}`
				m.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
					SecretString: &secretString,
				}, nil).Once()
			},
			expectedToken: "test_token",
		},
		// {
		// 	name: "config error",
		// 	mockConfig: func(m *MockConfigLoader) {
		// 		m.On("LoadDefaultConfig", mock.Anything, mock.Anything).Return(aws.Config{}, errors.New("config error")).Once()
		// 	},
		// 	expectedError: "error loading AWS config: config error",
		// },
		// {
		// 	name: "secret error",
		// 	mockConfig: func(m *MockConfigLoader) {
		// 		m.On("LoadDefaultConfig", mock.Anything, mock.Anything).Return(aws.Config{Region: "us-east-1"}, nil).Once()
		// 	},
		// 	mockSecret: func(m *MockSecretsManagerClient) {
		// 		m.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("secret error")).Once()
		// 	},
		// 	expectedError: "error fetching secret value: secret error",
		// },
		// {
		// 	name: "unmarshal error",
		// 	mockConfig: func(m *MockConfigLoader) {
		// 		m.On("LoadDefaultConfig", mock.Anything, mock.Anything).Return(aws.Config{Region: "us-east-1"}, nil).Once()
		// 	},
		// 	mockSecret: func(m *MockSecretsManagerClient) {
		// 		invalidSecretString := `{"INVALID_JSON"}`
		// 		m.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything).Return(&secretsmanager.GetSecretValueOutput{
		// 			SecretString: &invalidSecretString,
		// 		}, nil).Once()
		// 	},
		// 	expectedError: "error unmarshalling secret value: invalid character 'I' looking for beginning of object key string",
		// },
		{
			name: "cache hit",
			mockConfig: func(m *MockConfigLoader) {
				// No config load needed for cache hit
			},
			mockSecret: func(m *MockSecretsManagerClient) {
				// No secret fetch needed for cache hit
			},
			expectedToken: "cached_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new mocks for each test
			mockConfigLoader := new(MockConfigLoader)
			mockSecretsManagerClient := new(MockSecretsManagerClient)

			// Assign the new mocks to the global variables
			configLoader = mockConfigLoader
			secretsManagerClient = mockSecretsManagerClient

			// Set up the mocks for the current test case
			if tt.mockConfig != nil {
				tt.mockConfig(mockConfigLoader)
			}
			if tt.mockSecret != nil {
				tt.mockSecret(mockSecretsManagerClient)
			}

			// Pre-populate the cache for the "cache hit" test case
			if tt.name == "cache hit" {
				secretCache.Lock()
				secretCache.data["github_token"] = "cached_token"
				secretCache.Unlock()
			}

			// Call the function under test
			token, err := FetchSecretToken()

			// Check the results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}

			// Verify all expectations
			mockConfigLoader.AssertExpectations(t)
			mockSecretsManagerClient.AssertExpectations(t)
		})
	}
}
func TestFetchSecretTokenConfigError(t *testing.T) {
	// Backup the real instance
	realConfigLoader := configLoader

	// Restore the real instance after the test completes
	defer func() {
		configLoader = realConfigLoader
	}()

	// Create a new mock for this test
	mockConfigLoader := new(MockConfigLoader)

	// Assign the new mock to the global variable
	configLoader = mockConfigLoader

	// Set up the mock to return an error when loading the config
	mockConfigLoader.On("LoadDefaultConfig", mock.Anything, mock.Anything).Return(aws.Config{}, errors.New("config error")).Once()

	// Call the function under test
	token, err := FetchSecretToken()

	// Check the results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error loading AWS config: config error")
	assert.Empty(t, token)

	// Verify all expectations
	mockConfigLoader.AssertExpectations(t)
}
