package storage

import (
	"context"
)

// R2ClientInterface defines the interface that both real and mock R2 clients implement
type R2ClientInterface interface {
	ObjectExists(ctx context.Context, key string) (bool, error)
	Upload(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error)
	GetPublicURL(key string) string
	Delete(ctx context.Context, key string) error
}
