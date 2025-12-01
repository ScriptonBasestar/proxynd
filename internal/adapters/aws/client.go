package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// ClientConfig holds AWS client configuration
type ClientConfig struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

// AWSClient provides AWS service clients
type AWSClient struct {
	config    aws.Config
	ssmClient *ssm.Client
	s3Client  *s3.Client
}

// NewAWSClient creates a new AWS client with the provided configuration
func NewAWSClient(ctx context.Context, cfg ClientConfig) (*AWSClient, error) {
	// Load AWS configuration
	var awsConfig aws.Config
	var err error

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		// Use provided credentials
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(
					cfg.AccessKeyID,
					cfg.SecretAccessKey,
					cfg.SessionToken,
				),
			)),
		)
	} else {
		// Use default credential chain (IAM roles, env vars, etc.)
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := &AWSClient{
		config:    awsConfig,
		ssmClient: ssm.NewFromConfig(awsConfig),
		s3Client:  s3.NewFromConfig(awsConfig),
	}

	return client, nil
}

// GetSSMClient returns the SSM client for parameter store operations
func (c *AWSClient) GetSSMClient() *ssm.Client {
	return c.ssmClient
}

// GetS3Client returns the S3 client for storage operations
func (c *AWSClient) GetS3Client() *s3.Client {
	return c.s3Client
}

// GetParameter retrieves a parameter from AWS SSM Parameter Store
func (c *AWSClient) GetParameter(ctx context.Context, name string, withDecryption bool) (string, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(withDecryption),
	}

	result, err := c.ssmClient.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get parameter %s: %w", name, err)
	}

	if result.Parameter == nil || result.Parameter.Value == nil {
		return "", fmt.Errorf("parameter %s not found or has no value", name)
	}

	return *result.Parameter.Value, nil
}

// GetParameters retrieves multiple parameters from AWS SSM Parameter Store
func (c *AWSClient) GetParameters(ctx context.Context, names []string, withDecryption bool) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	input := &ssm.GetParametersInput{
		Names:          names,
		WithDecryption: aws.Bool(withDecryption),
	}

	result, err := c.ssmClient.GetParameters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get parameters: %w", err)
	}

	// Convert result to map
	params := make(map[string]string)
	for _, param := range result.Parameters {
		if param.Name != nil && param.Value != nil {
			params[*param.Name] = *param.Value
		}
	}

	// Check for invalid parameters
	if len(result.InvalidParameters) > 0 {
		return params, fmt.Errorf("invalid parameters found: %v", result.InvalidParameters)
	}

	return params, nil
}

// GetParametersByPath retrieves all parameters under a specific path
// This is useful for retrieving configuration hierarchies
func (c *AWSClient) GetParametersByPath(ctx context.Context, path string, withDecryption, recursive bool) (map[string]string, error) {
	params := make(map[string]string)
	var nextToken *string

	for {
		input := &ssm.GetParametersByPathInput{
			Path:           aws.String(path),
			WithDecryption: aws.Bool(withDecryption),
			Recursive:      aws.Bool(recursive),
			NextToken:      nextToken,
		}

		result, err := c.ssmClient.GetParametersByPath(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get parameters by path %s: %w", path, err)
		}

		// Add parameters to map
		for _, param := range result.Parameters {
			if param.Name != nil && param.Value != nil {
				params[*param.Name] = *param.Value
			}
		}

		// Check if there are more parameters
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return params, nil
}

// PutParameter creates or updates a parameter in AWS SSM Parameter Store
func (c *AWSClient) PutParameter(ctx context.Context, name, value string, paramType types.ParameterType, overwrite bool) error {
	input := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      paramType,
		Overwrite: aws.Bool(overwrite),
	}

	_, err := c.ssmClient.PutParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put parameter %s: %w", name, err)
	}

	return nil
}

// DeleteParameter deletes a parameter from AWS SSM Parameter Store
func (c *AWSClient) DeleteParameter(ctx context.Context, name string) error {
	input := &ssm.DeleteParameterInput{
		Name: aws.String(name),
	}

	_, err := c.ssmClient.DeleteParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete parameter %s: %w", name, err)
	}

	return nil
}

// HealthCheck verifies that the AWS client can connect to AWS services
func (c *AWSClient) HealthCheck(ctx context.Context) error {
	// Try to describe parameters to verify SSM connectivity
	_, err := c.ssmClient.DescribeParameters(ctx, &ssm.DescribeParametersInput{
		MaxResults: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("AWS health check failed: %w", err)
	}

	return nil
}
