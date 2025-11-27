package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAWSClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
	}{
		{
			name: "with static credentials",
			cfg: ClientConfig{
				Region:          "us-east-1",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErr: false,
		},
		{
			name: "with region only (default credentials)",
			cfg: ClientConfig{
				Region: "us-west-2",
			},
			wantErr: false,
		},
		{
			name: "with session token",
			cfg: ClientConfig{
				Region:          "eu-west-1",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "FwoGZXIvYXdzEBQaDKExAMPLETOKEN",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client, err := NewAWSClient(ctx, tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.NotNil(t, client.GetSSMClient())
				assert.NotNil(t, client.GetS3Client())
			}
		})
	}
}

func TestAWSClient_GetSSMClient(t *testing.T) {
	ctx := context.Background()
	cfg := ClientConfig{
		Region:          "us-east-1",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	ssmClient := client.GetSSMClient()
	assert.NotNil(t, ssmClient, "SSM client should not be nil")
}

func TestAWSClient_GetS3Client(t *testing.T) {
	ctx := context.Background()
	cfg := ClientConfig{
		Region:          "us-east-1",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	s3Client := client.GetS3Client()
	assert.NotNil(t, s3Client, "S3 client should not be nil")
}

// Integration tests (require real AWS credentials and SSM parameters)
// These are skipped in unit tests, run with -tags=integration

func TestAWSClient_GetParameter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
		// Credentials from environment or IAM role
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)

	// Note: This test requires a real SSM parameter named /test/param
	// Create it manually or skip this test
	t.Run("get existing parameter", func(t *testing.T) {
		t.Skip("Requires real AWS SSM parameter /test/param")

		value, err := client.GetParameter(ctx, "/test/param", false)
		require.NoError(t, err)
		assert.NotEmpty(t, value)
	})

	t.Run("get non-existent parameter", func(t *testing.T) {
		t.Skip("Requires real AWS credentials")

		_, err := client.GetParameter(ctx, "/non/existent/param/12345", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get parameter")
	})
}

func TestAWSClient_GetParameters_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)

	t.Run("get multiple parameters", func(t *testing.T) {
		t.Skip("Requires real AWS SSM parameters")

		names := []string{"/test/param1", "/test/param2"}
		params, err := client.GetParameters(ctx, names, false)
		require.NoError(t, err)
		assert.Len(t, params, 2)
	})

	t.Run("empty parameter list", func(t *testing.T) {
		params, err := client.GetParameters(ctx, []string{}, false)
		require.NoError(t, err)
		assert.Empty(t, params)
	})

	t.Run("with invalid parameters", func(t *testing.T) {
		t.Skip("Requires real AWS credentials")

		names := []string{"/test/param1", "/invalid/param"}
		_, err := client.GetParameters(ctx, names, false)
		// Should return partial results with error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parameters")
	})
}

func TestAWSClient_GetParametersByPath_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)

	t.Run("get parameters by path", func(t *testing.T) {
		t.Skip("Requires real AWS SSM parameters under /test/")

		params, err := client.GetParametersByPath(ctx, "/test", false, false)
		require.NoError(t, err)
		assert.NotEmpty(t, params)
	})

	t.Run("get parameters recursively", func(t *testing.T) {
		t.Skip("Requires real AWS SSM parameters under /test/")

		params, err := client.GetParametersByPath(ctx, "/test", false, true)
		require.NoError(t, err)
		assert.NotEmpty(t, params)
	})
}

func TestAWSClient_PutParameter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)

	t.Run("create string parameter", func(t *testing.T) {
		t.Skip("Requires AWS write permissions")

		err := client.PutParameter(ctx, "/test/unit-test", "test-value", types.ParameterTypeString, true)
		require.NoError(t, err)

		// Verify parameter was created
		value, err := client.GetParameter(ctx, "/test/unit-test", false)
		require.NoError(t, err)
		assert.Equal(t, "test-value", value)

		// Cleanup
		_ = client.DeleteParameter(ctx, "/test/unit-test")
	})

	t.Run("create secure string parameter", func(t *testing.T) {
		t.Skip("Requires AWS write permissions")

		err := client.PutParameter(ctx, "/test/secure-test", "secret-value", types.ParameterTypeSecureString, true)
		require.NoError(t, err)

		// Verify parameter was created
		value, err := client.GetParameter(ctx, "/test/secure-test", true)
		require.NoError(t, err)
		assert.Equal(t, "secret-value", value)

		// Cleanup
		_ = client.DeleteParameter(ctx, "/test/secure-test")
	})
}

func TestAWSClient_DeleteParameter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)

	t.Run("delete existing parameter", func(t *testing.T) {
		t.Skip("Requires AWS write permissions")

		// Create parameter
		err := client.PutParameter(ctx, "/test/delete-test", "test-value", types.ParameterTypeString, true)
		require.NoError(t, err)

		// Delete parameter
		err = client.DeleteParameter(ctx, "/test/delete-test")
		require.NoError(t, err)

		// Verify deletion
		_, err = client.GetParameter(ctx, "/test/delete-test", false)
		assert.Error(t, err)
	})
}

func TestAWSClient_HealthCheck_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(t, err)

	t.Run("health check passes", func(t *testing.T) {
		t.Skip("Requires AWS credentials")

		err := client.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}

// Benchmark tests
func BenchmarkAWSClient_GetParameter(b *testing.B) {
	b.Skip("Requires real AWS SSM parameter")

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetParameter(ctx, "/test/benchmark", false)
	}
}

func BenchmarkAWSClient_GetParameters(b *testing.B) {
	b.Skip("Requires real AWS SSM parameters")

	ctx := context.Background()
	cfg := ClientConfig{
		Region: "us-east-1",
	}

	client, err := NewAWSClient(ctx, cfg)
	require.NoError(b, err)

	names := []string{"/test/param1", "/test/param2", "/test/param3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetParameters(ctx, names, false)
	}
}
