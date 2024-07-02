package gitsetup

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type mockConfigLoader struct{}

func (m *mockConfigLoader) LoadDefaultConfig(ctx context.Context, options ...func(*config.LoadOptions) error) (aws.Config, error) {
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
