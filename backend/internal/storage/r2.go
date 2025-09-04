package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type R2Client struct {
	client          *s3.Client
	bucket          string
	publicBaseURL   string
}

type UploadResult struct {
	Key         string
	URL         string
	ETag        string
	Size        int64
	ContentType string
}

func NewR2Client(ctx context.Context, accountID, accessKeyID, secretAccessKey, bucket, endpoint, publicBaseURL string) (*R2Client, error) {
	// Create custom credentials
	creds := credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")
	
	// Create AWS config for R2
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: endpoint,
				}, nil
			})),
		config.WithRegion("auto"), // R2 uses "auto" as region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	return &R2Client{
		client:        client,
		bucket:        bucket,
		publicBaseURL: strings.TrimSuffix(publicBaseURL, "/"),
	}, nil
}

// ObjectExists checks if an object exists in R2
func (r *R2Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	
	if err != nil {
		// For 404 errors (object doesn't exist), return false without error
		if strings.Contains(err.Error(), "404") || 
		   strings.Contains(err.Error(), "NotFound") || 
		   strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, err
	}
	
	return true, nil
}

// Upload uploads data to R2 with the specified key
func (r *R2Client) Upload(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
		Metadata: map[string]string{
			"source": "format.hackclub.com",
		},
	}

	result, err := r.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to R2: %v", err)
	}

	return &UploadResult{
		Key:         key,
		URL:         r.GetPublicURL(key),
		ETag:        aws.ToString(result.ETag),
		Size:        int64(len(data)),
		ContentType: contentType,
	}, nil
}

// GetPublicURL returns the public CDN URL for the given key
func (r *R2Client) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/%s", r.publicBaseURL, key)
}

// Delete removes an object from R2
func (r *R2Client) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	return err
}

// GetObjectMetadata retrieves metadata for an object
func (r *R2Client) GetObjectMetadata(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	return r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
}

// ListObjects lists objects with the given prefix
func (r *R2Client) ListObjects(ctx context.Context, prefix string, maxKeys int32) ([]types.Object, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys,
	}

	result, err := r.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.Contents, nil
}
