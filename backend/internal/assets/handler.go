package assets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hackclub/format/internal/session"
	"github.com/rs/zerolog"
)

type Handler struct {
	service *Service
	logger  zerolog.Logger
}

func NewHandler(service *Service, logger zerolog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// HandleUpload handles single file upload or URL/data URI processing
func (h *Handler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if this is multipart form data (file upload) or JSON
	contentType := r.Header.Get("Content-Type")
	
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle file upload
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to parse multipart form")
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}

		asset, err := h.service.ProcessFromData(ctx, &ProcessInput{
			Data:        data,
			ContentType: http.DetectContentType(data),
			SourceURL:   "upload",
		})
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to process uploaded file")
			http.Error(w, fmt.Sprintf("Failed to process image: %v", err), http.StatusInternalServerError)
			return
		}

		h.writeJSONResponse(w, asset)
		return
	}

	// Handle JSON request (URL or data URI)
	var req struct {
		URL     string `json:"url,omitempty"`
		DataURI string `json:"dataUri,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	var asset *Asset
	var err error

	switch {
	case req.URL != "":
		asset, err = h.service.ProcessFromURL(ctx, req.URL)
	case req.DataURI != "":
		asset, err = h.service.ProcessFromDataURI(ctx, req.DataURI)
	default:
		http.Error(w, "Either 'url' or 'dataUri' must be provided", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error().Err(err).Str("url", req.URL).Msg("failed to process image")
		http.Error(w, fmt.Sprintf("Failed to process image: %v", err), http.StatusInternalServerError)
		return
	}

	h.writeJSONResponse(w, asset)
}

// HandleBatch handles batch processing of multiple images
func (h *Handler) HandleBatch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Items []BatchInput `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "No items provided", http.StatusBadRequest)
		return
	}

	// Limit batch size
	maxBatchSize := 20
	if len(req.Items) > maxBatchSize {
		http.Error(w, fmt.Sprintf("Batch size too large (max %d)", maxBatchSize), http.StatusBadRequest)
		return
	}

	assets, err := h.service.ProcessBatch(ctx, req.Items)
	if err != nil {
		h.logger.Error().Err(err).Int("batch_size", len(req.Items)).Msg("failed to process batch")
		http.Error(w, fmt.Sprintf("Failed to process batch: %v", err), http.StatusInternalServerError)
		return
	}

	h.writeJSONResponse(w, map[string]interface{}{
		"assets": assets,
		"count":  len(assets),
	})
}

// HandleGetAsset handles retrieving asset metadata by ID/key
func (h *Handler) HandleGetAsset(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/assets/")
	if path == "" {
		http.Error(w, "Asset ID required", http.StatusBadRequest)
		return
	}

	// For now, just return a simple response
	// In a full implementation, you'd look up the asset metadata from storage
	h.writeJSONResponse(w, map[string]string{
		"message": "Asset metadata endpoint - not fully implemented",
		"id":      path,
	})
}

func (h *Handler) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error().Err(err).Msg("failed to encode JSON response")
	}
}

// Middleware for rate limiting (simple in-memory implementation)
func (h *Handler) RateLimit(next http.Handler) http.Handler {
	// This is a placeholder for rate limiting
	// In production, you'd use a proper rate limiter like golang.org/x/time/rate
	return next
}

// getUserFromSession is a helper to get user from session
func (h *Handler) getUserFromSession(r *http.Request) *session.User {
	ctx := r.Context()
	if user := ctx.Value("user"); user != nil {
		if u, ok := user.(*session.User); ok {
			return u
		}
	}
	return nil
}
