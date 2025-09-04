package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// MockR2Client provides a local filesystem mock of R2Client for development
type MockR2Client struct {
	baseDir       string
	publicBaseURL string
}

func NewMockR2Client(baseDir, publicBaseURL string) *MockR2Client {
	// Ensure the base directory exists
	os.MkdirAll(baseDir, 0755)
	
	return &MockR2Client{
		baseDir:       baseDir,
		publicBaseURL: publicBaseURL,
	}
}

// ObjectExists checks if a file exists locally
func (m *MockR2Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	filePath := filepath.Join(m.baseDir, key)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Upload saves data to local filesystem
func (m *MockR2Client) Upload(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	filePath := filepath.Join(m.baseDir, key)
	
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}
	
	return &UploadResult{
		Key:         key,
		URL:         m.GetPublicURL(key),
		ETag:        fmt.Sprintf(`"%x"`, data), // Simple ETag
		Size:        int64(len(data)),
		ContentType: contentType,
	}, nil
}

// GetPublicURL returns the public URL for a file
func (m *MockR2Client) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/%s", m.publicBaseURL, key)
}

// Additional methods to match interface
func (m *MockR2Client) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(m.baseDir, key)
	return os.Remove(filePath)
}
