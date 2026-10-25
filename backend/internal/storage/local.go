package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// LocalStorage implements StorageProvider for local filesystem storage
type LocalStorage struct {
	BaseDir string
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &LocalStorage{BaseDir: baseDir}, nil
}

// Upload saves a file to local storage
func (ls *LocalStorage) Upload(ctx context.Context, fileBytes []byte, fileName string) (string, error) {
	key, err := newObjectKey()
	if err != nil {
		return "", err
	}
	
	// Construct full file path
	fullPath := filepath.Join(ls.BaseDir, key)
	
	// Create directory structure if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Never truncate an existing object, even in the unlikely event of a collision.
	file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	_, writeErr := file.Write(fileBytes)
	closeErr := file.Close()
	if writeErr != nil {
		return "", fmt.Errorf("failed to write file: %w", writeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("failed to close file: %w", closeErr)
	}
	
	return key, nil
}

// ReadText reads a text file from local storage
func (ls *LocalStorage) ReadText(ctx context.Context, fileKey string) (string, error) {
	data, err := ls.Download(ctx, fileKey)
	if err != nil {
		return "", err
	}

	content := string(data)
	if !utf8.ValidString(content) {
		return "", fmt.Errorf("invalid UTF-8 content")
	}
	return content, nil
}

// Download reads a file from local storage and returns its bytes
func (ls *LocalStorage) Download(ctx context.Context, fileKey string) ([]byte, error) {
	// Sanitize the file key to prevent directory traversal
	sanitizedKey := strings.ReplaceAll(fileKey, "..", "")
	
	fullPath := filepath.Join(ls.BaseDir, sanitizedKey)
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return data, nil
}

// Delete removes a file from local storage
func (ls *LocalStorage) Delete(ctx context.Context, fileKey string) error {
	// Sanitize the file key to prevent directory traversal
	sanitizedKey := strings.ReplaceAll(fileKey, "..", "")
	
	fullPath := filepath.Join(ls.BaseDir, sanitizedKey)
	
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	return nil
}
