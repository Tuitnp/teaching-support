package storage

import (
	"bytes"
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	Client     *s3.Client
	BucketName string
}

func NewS3Storage(client *s3.Client, bucket string) *S3Storage {
	return &S3Storage{Client: client, BucketName: bucket}
}

func (s *S3Storage) Upload(ctx context.Context, fileBytes []byte, fileName string) (string, error) {
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
	return key, err
}

func (s *S3Storage) ReadText(ctx context.Context, fileKey string) (string, error) {
	data, err := s.Download(ctx, fileKey)
	if err != nil {
		return "", err
	}

	content := string(data)
	if !utf8.ValidString(content) {
		return "", fmt.Errorf("invalid UTF-8 content")
	}
	return content, nil
}

func (s *S3Storage) Download(ctx context.Context, fileKey string) ([]byte, error) {
	result, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(result.Body)
	return buf.Bytes(), err
}

func (s *S3Storage) Delete(ctx context.Context, fileKey string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.BucketName),
		Key:    aws.String(fileKey),
	})
	return err
}

// isValidUTF8 checks if the byte slice contains only valid UTF-8 characters
func isValidUTF8(data []byte) bool {
	// Convert bytes to string - Go will replace invalid UTF-8 sequences with the replacement rune
	content := string(data)
	
	// Check if any replacement runes exist (0xfffd indicates invalid UTF-8)
	for _, r := range content {
		if r == 0xfffd {
			return false
		}
	}
	return true
}
