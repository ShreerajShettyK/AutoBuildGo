package ecr

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/stretchr/testify/assert"
)

// MockECRClient is a mock implementation of ECRClientInterface for testing.
type MockECRClient struct {
	CreateRepositoryFunc func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
}

// CreateRepository mocks the CreateRepository method.
func (m *MockECRClient) CreateRepository(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
	if m.CreateRepositoryFunc != nil {
		return m.CreateRepositoryFunc(ctx, params, optFns...)
	}
	return nil, nil
}

func TestCreateRepo(t *testing.T) {
	// Positive test case
	t.Run("CreateRepository_Success", func(t *testing.T) {
		mockClient := &MockECRClient{
			CreateRepositoryFunc: func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
				return &ecr.CreateRepositoryOutput{}, nil
			},
		}
		err := CreateRepo("testRepo", mockClient)
		assert.NoError(t, err)
	})

	// Negative test case: Generic failure
	t.Run("CreateRepository_Failure", func(t *testing.T) {
		mockClient := &MockECRClient{
			CreateRepositoryFunc: func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
				return nil, errors.New("some error message") // Replace this with the error you want to simulate
			},
		}
		err := CreateRepo("testRepo", mockClient)
		assert.Error(t, err)
	})

	// Negative test case: Repository already exists
	t.Run("CreateRepository_RepoAlreadyExists", func(t *testing.T) {
		mockClient := &MockECRClient{
			CreateRepositoryFunc: func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
				return nil, errors.New("repository already exists") // Simulate repository already exists error
			},
		}
		err := CreateRepo("testRepo", mockClient)
		assert.Error(t, err)
	})
}

func TestNewClient(t *testing.T) {
	cfg := aws.Config{}
	client := NewClient(cfg)
	assert.NotNil(t, client)
}

func TestLoadAWSConfig(t *testing.T) {
	t.Run("LoadAWSConfig_Success", func(t *testing.T) {
		_, err := LoadAWSConfig()
		assert.NoError(t, err)
	})

	// t.Run("LoadAWSConfig_Failure", func(t *testing.T) {
	// 	// Simulate failure by replacing the config loader temporarily
	// 	originalConfigLoader := config.LoadDefaultConfig
	// 	defer func() { config.LoadDefaultConfig = originalConfigLoader }()

	// 	config.LoadDefaultConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	// 		return aws.Config{}, errors.New("mock error loading config")
	// 	}

	// 	_, err := LoadAWSConfig()
	// 	assert.Error(t, err)
	// })
}
