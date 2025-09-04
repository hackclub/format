package assets

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/hackclub/format/internal/imageproc"
	"github.com/hackclub/format/internal/storage"
	"github.com/hackclub/format/internal/util"
	"github.com/rs/zerolog"
)

type Service struct {
	processor *imageproc.Processor
	storage   *storage.R2Client
	fetcher   *util.HTTPFetcher
	logger    zerolog.Logger
}

type Asset struct {
	URL         string `json:"url"`
	MIME        string `json:"mime"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Bytes       int    `json:"bytes"`
	Hash        string `json:"hash"`
	Deduped     bool   `json:"deduped"`
	Key         string `json:"key,omitempty"`
}

type ProcessInput struct {
	Data        []byte
	ContentType string
	SourceURL   string
}

func NewService(processor *imageproc.Processor, storage *storage.R2Client, logger zerolog.Logger) *Service {
	return &Service{
		processor: processor,
		storage:   storage,
		fetcher:   util.NewHTTPFetcher(),
		logger:    logger,
	}
}

// ProcessFromURL processes an image from a URL
func (s *Service) ProcessFromURL(ctx context.Context, imageURL string) (*Asset, error) {
	s.logger.Info().Str("url", imageURL).Msg("processing image from URL")

	// Fetch the image
	data, contentType, err := s.fetcher.FetchURL(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %v", err)
	}

	return s.ProcessFromData(ctx, &ProcessInput{
		Data:        data,
		ContentType: contentType,
		SourceURL:   imageURL,
	})
}

// ProcessFromDataURI processes an image from a data URI
func (s *Service) ProcessFromDataURI(ctx context.Context, dataURI string) (*Asset, error) {
	s.logger.Info().Str("dataURI", dataURI[:min(100, len(dataURI))]).Msg("processing image from data URI")

	// Parse data URI
	data, contentType, err := s.parseDataURI(dataURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data URI: %v", err)
	}

	return s.ProcessFromData(ctx, &ProcessInput{
		Data:        data,
		ContentType: contentType,
		SourceURL:   "data:",
	})
}

// ProcessFromData processes raw image data
func (s *Service) ProcessFromData(ctx context.Context, input *ProcessInput) (*Asset, error) {
	// Process the image
	result, err := s.processor.Process(input.Data, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %v", err)
	}

	// Calculate hash for deduplication
	hash := sha256.Sum256(result.Data)
	hashStr := fmt.Sprintf("%x", hash)

	// Generate key
	ext := util.GetImageExtension(result.ContentType)
	key := util.Base32Key(result.Data, ext)

	s.logger.Info().
		Str("hash", hashStr[:16]).
		Str("key", key).
		Int("original_size", result.OriginalSize).
		Int("compressed_size", result.CompressedSize).
		Msg("processed image")

	// Check if object already exists (deduplication)
	exists, err := s.storage.ObjectExists(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to check if object exists: %v", err)
	}

	var publicURL string
	deduped := false

	if exists {
		// Object already exists, just return the URL
		publicURL = s.storage.GetPublicURL(key)
		deduped = true
		s.logger.Info().Str("key", key).Str("public_url", publicURL).Msg("object already exists, using existing")
	} else {
		// Upload new object
		uploadResult, err := s.storage.Upload(ctx, key, result.Data, result.ContentType)
		if err != nil {
			return nil, fmt.Errorf("failed to upload to storage: %v", err)
		}
		publicURL = uploadResult.URL
		s.logger.Info().Str("key", key).Str("upload_url", uploadResult.URL).Str("public_url", publicURL).Msg("uploaded new object")
	}

	return &Asset{
		URL:     publicURL,
		MIME:    result.ContentType,
		Width:   result.Width,
		Height:  result.Height,
		Bytes:   result.CompressedSize,
		Hash:    "sha256:" + hashStr,
		Deduped: deduped,
		Key:     key,
	}, nil
}

// ProcessBatch processes multiple images
func (s *Service) ProcessBatch(ctx context.Context, inputs []BatchInput) ([]*Asset, error) {
	assets := make([]*Asset, 0, len(inputs))
	
	for i, input := range inputs {
		s.logger.Info().Int("index", i).Msg("processing batch item")
		
		var asset *Asset
		var err error
		
		switch {
		case input.URL != "":
			asset, err = s.ProcessFromURL(ctx, input.URL)
		case input.DataURI != "":
			asset, err = s.ProcessFromDataURI(ctx, input.DataURI)
		case len(input.Data) > 0:
			asset, err = s.ProcessFromData(ctx, &ProcessInput{
				Data:        input.Data,
				ContentType: input.ContentType,
				SourceURL:   "upload",
			})
		default:
			err = fmt.Errorf("no valid input provided for batch item %d", i)
		}
		
		if err != nil {
			s.logger.Error().Err(err).Int("index", i).Msg("failed to process batch item")
			return nil, fmt.Errorf("failed to process item %d: %v", i, err)
		}
		
		assets = append(assets, asset)
	}
	
	return assets, nil
}

type BatchInput struct {
	URL         string `json:"url,omitempty"`
	DataURI     string `json:"dataUri,omitempty"`
	Data        []byte `json:"-"` // For file uploads
	ContentType string `json:"-"`
}

func (s *Service) parseDataURI(dataURI string) ([]byte, string, error) {
	// Parse data URI format: data:[<mediatype>][;base64],<data>
	if !strings.HasPrefix(dataURI, "data:") {
		return nil, "", fmt.Errorf("invalid data URI format")
	}

	// Remove "data:" prefix
	content := dataURI[5:]

	// Find the comma separator
	commaIndex := strings.Index(content, ",")
	if commaIndex == -1 {
		return nil, "", fmt.Errorf("invalid data URI: missing comma separator")
	}

	// Parse header and data
	header := content[:commaIndex]
	encodedData := content[commaIndex+1:]

	// Parse content type and encoding
	var contentType string
	isBase64 := false

	parts := strings.Split(header, ";")
	if len(parts) > 0 && parts[0] != "" {
		contentType = parts[0]
	} else {
		contentType = "text/plain" // Default
	}

	for _, part := range parts[1:] {
		if part == "base64" {
			isBase64 = true
		}
	}

	// Decode data
	var data []byte
	var err error

	if isBase64 {
		data, err = base64.StdEncoding.DecodeString(encodedData)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64 data: %v", err)
		}
	} else {
		// URL decode
		decoded, err := url.QueryUnescape(encodedData)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode URL data: %v", err)
		}
		data = []byte(decoded)
	}

	return data, contentType, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
