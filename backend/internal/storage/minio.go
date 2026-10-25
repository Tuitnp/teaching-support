package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MinIOStorage implements StorageProvider interface for MinIO (S3-compatible)
type MinIOStorage struct {
	Client     *s3.Client
	BucketName string
}

// NewMinIOStorage creates a new MinIO storage provider
// endpoint: MinIO server URL (e.g., "http://localhost:9000")
// accessKey: MinIO access key
// secretKey: MinIO secret key
// bucket: Bucket name to use
// useSSL: Set to true if using HTTPS
func NewMinIOStorage(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOStorage, error) {
	ctx := context.TODO()

	// Load config (no resolver here)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			}, nil
		})),
		config.WithRegion("us-east-1"), // required but arbitrary for MinIO
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load MinIO config: %w", err)
	}

	// Create S3 client with MinIO endpoint
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	storage := &MinIOStorage{
		Client:     client,
		BucketName: bucket,
	}

	// Ensure bucket exists
	if err := storage.ensureBucket(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return storage, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (s *MinIOStorage) ensureBucket(ctx context.Context) error {
    _, err := s.Client.CreateBucket(ctx, &s3.CreateBucketInput{
        Bucket: aws.String(s.BucketName),
    })
    if err != nil {
        var bucketAlreadyExists *s3types.BucketAlreadyExists
        var bucketAlreadyOwnedByYou *s3types.BucketAlreadyOwnedByYou

        if errors.As(err, &bucketAlreadyExists) ||
            errors.As(err, &bucketAlreadyOwnedByYou) {
            return nil // Bucket already exists, that's fine
        }
        return fmt.Errorf("failed to create bucket: %w", err)
    }
    return nil
}

// Upload uploads a file to MinIO
func (s *MinIOStorage) Upload(ctx context.Context, fileBytes []byte, fileName string) (string, error) {
	key, err := newObjectKey()
	if err != nil {
		return "", err
	}
	uploader := manager.NewUploader(s.Client)

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileBytes),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to MinIO: %w", err)
	}

	return key, nil
}

// ReadText reads a text file from MinIO and returns its content as a string
func (s *MinIOStorage) ReadText(ctx context.Context, fileKey string) (string, error) {
	data, err := s.Download(ctx, fileKey)
	if err != nil {
		return "", err
	}

	// Validate that the content is valid UTF-8 text
	if !isValidUTF8(data) {
		return "", fmt.Errorf("invalid file content: file is not valid text")
	}

	return string(data), nil
}

// Download downloads a file from MinIO and returns its content as bytes
func (s *MinIOStorage) Download(ctx context.Context, fileKey string) ([]byte, error) {
	result, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return buf.Bytes(), nil
}

// Delete deletes a file from MinIO
func (s *MinIOStorage) Delete(ctx context.Context, fileKey string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from MinIO: %w", err)
	}

	return nil
}
