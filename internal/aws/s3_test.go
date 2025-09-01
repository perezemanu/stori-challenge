package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseS3URL(t *testing.T) {
	tests := []struct {
		name        string
		s3URL       string
		bucket      string
		key         string
		expectError bool
	}{
		{
			name:        "valid S3 URL",
			s3URL:       "s3://my-bucket/path/to/file.csv",
			bucket:      "my-bucket",
			key:         "path/to/file.csv",
			expectError: false,
		},
		{
			name:        "S3 URL with single path",
			s3URL:       "s3://my-bucket/file.csv",
			bucket:      "my-bucket",
			key:         "file.csv",
			expectError: false,
		},
		{
			name:        "S3 URL with deep path",
			s3URL:       "s3://my-bucket/input/2023/12/31/transactions.csv",
			bucket:      "my-bucket",
			key:         "input/2023/12/31/transactions.csv",
			expectError: false,
		},
		{
			name:        "invalid URL - not S3",
			s3URL:       "https://example.com/file.csv",
			bucket:      "",
			key:         "",
			expectError: true,
		},
		{
			name:        "invalid URL - missing key",
			s3URL:       "s3://my-bucket",
			bucket:      "",
			key:         "",
			expectError: true,
		},
		{
			name:        "invalid URL - missing bucket",
			s3URL:       "s3:///file.csv",
			bucket:      "",
			key:         "",
			expectError: true,
		},
		{
			name:        "empty URL",
			s3URL:       "",
			bucket:      "",
			key:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := ParseS3URL(tt.s3URL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, bucket)
				assert.Empty(t, key)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.bucket, bucket)
				assert.Equal(t, tt.key, key)
			}
		})
	}
}

func TestGenerateProcessedKey(t *testing.T) {
	tests := []struct {
		name        string
		originalKey string
		expected    string
	}{
		{
			name:        "input directory key",
			originalKey: "input/transactions.csv",
			expected:    "processed/transactions.csv",
		},
		{
			name:        "key without input prefix",
			originalKey: "transactions.csv",
			expected:    "processed/transactions.csv",
		},
		{
			name:        "nested input path",
			originalKey: "input/2023/12/31/transactions.csv",
			expected:    "processed/2023/12/31/transactions.csv",
		},
		{
			name:        "already processed key",
			originalKey: "processed/transactions.csv",
			expected:    "processed/processed/transactions.csv",
		},
		{
			name:        "empty key",
			originalKey: "",
			expected:    "processed/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateProcessedKey(tt.originalKey)
			assert.Equal(t, tt.expected, result)
		})
	}
}
