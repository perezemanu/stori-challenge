package aws

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// S3Client provides operations for interacting with AWS S3
type S3Client struct {
	client *s3.Client
	logger *zap.Logger
}

// NewS3Client creates a new S3 client
func NewS3Client(ctx context.Context, logger *zap.Logger) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Client{
		client: client,
		logger: logger,
	}, nil
}

// GetObject retrieves an object from S3 and returns a reader
func (c *S3Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	c.logger.Info("Getting S3 object",
		zap.String("bucket", bucket),
		zap.String("key", key),
	)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.client.GetObject(ctx, input)
	if err != nil {
		c.logger.Error("Failed to get S3 object",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get object s3://%s/%s: %w", bucket, key, err)
	}

	c.logger.Info("Successfully retrieved S3 object",
		zap.String("bucket", bucket),
		zap.String("key", key),
		zap.Int64("content_length", aws.ToInt64(result.ContentLength)),
	)

	return result.Body, nil
}

// MoveObject moves an object from one key to another within the same bucket
func (c *S3Client) MoveObject(ctx context.Context, bucket, sourceKey, destKey string) error {
	c.logger.Info("Moving S3 object",
		zap.String("bucket", bucket),
		zap.String("source_key", sourceKey),
		zap.String("dest_key", destKey),
	)

	// First, copy the object to the new location
	copySource := fmt.Sprintf("%s/%s", bucket, sourceKey)
	copyInput := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(destKey),
		CopySource: aws.String(copySource),
	}

	_, err := c.client.CopyObject(ctx, copyInput)
	if err != nil {
		c.logger.Error("Failed to copy S3 object",
			zap.String("bucket", bucket),
			zap.String("source_key", sourceKey),
			zap.String("dest_key", destKey),
			zap.Error(err),
		)
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// Then, delete the original object
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(sourceKey),
	}

	_, err = c.client.DeleteObject(ctx, deleteInput)
	if err != nil {
		c.logger.Error("Failed to delete original S3 object",
			zap.String("bucket", bucket),
			zap.String("source_key", sourceKey),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete original object: %w", err)
	}

	c.logger.Info("Successfully moved S3 object",
		zap.String("bucket", bucket),
		zap.String("source_key", sourceKey),
		zap.String("dest_key", destKey),
	)

	return nil
}

// GetObjectSize returns the size of an object in S3
func (c *S3Client) GetObjectSize(ctx context.Context, bucket, key string) (int64, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to head object s3://%s/%s: %w", bucket, key, err)
	}

	return aws.ToInt64(result.ContentLength), nil
}

// ObjectExists checks if an object exists in S3
func (c *S3Client) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.GetObjectSize(ctx, bucket, key)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ParseS3URL parses an S3 URL and returns bucket and key
func ParseS3URL(s3URL string) (bucket, key string, err error) {
	if s3URL == "" {
		return "", "", fmt.Errorf("S3 URL cannot be empty")
	}

	if !strings.HasPrefix(s3URL, "s3://") {
		return "", "", fmt.Errorf("invalid S3 URL format: %s", s3URL)
	}

	// Remove s3:// prefix
	path := strings.TrimPrefix(s3URL, "s3://")

	// Split bucket and key
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid S3 URL format, missing key: %s", s3URL)
	}

	bucket = parts[0]
	key = parts[1]

	// Validate bucket name is not empty
	if bucket == "" {
		return "", "", fmt.Errorf("invalid S3 URL format, missing bucket: %s", s3URL)
	}

	// Validate key is not empty
	if key == "" {
		return "", "", fmt.Errorf("invalid S3 URL format, missing key: %s", s3URL)
	}

	return bucket, key, nil
}

// GenerateProcessedKey generates a key for the processed directory
func GenerateProcessedKey(originalKey string) string {
	// Remove "input/" prefix if present
	key := strings.TrimPrefix(originalKey, "input/")

	// Add to processed directory with timestamp-based structure
	// Example: input/transactions.csv -> processed/2023/12/31/transactions.csv
	return fmt.Sprintf("processed/%s", key)
}
