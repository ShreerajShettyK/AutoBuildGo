package ecr

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type AWSCredentialsRetriever interface {
	Retrieve(ctx context.Context) (aws.Credentials, error)
}

var loadDefaultConfig = config.LoadDefaultConfig

// GetAWSCredentials retrieves AWS credentials
func GetAWSCredentials() (AWSCredentials, error) {
	cfg, err := loadDefaultConfig(context.Background())
	if err != nil {
		return AWSCredentials{}, errors.New("failed to load AWS config")
	}

	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		return AWSCredentials{}, errors.New("failed to retrieve AWS credentials")
	}

	return AWSCredentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
	}, nil
}
