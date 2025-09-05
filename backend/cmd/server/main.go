package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hackclub/format/internal/assets"
	"github.com/hackclub/format/internal/auth"
	"github.com/hackclub/format/internal/config"
	"github.com/hackclub/format/internal/html"
	httphandler "github.com/hackclub/format/internal/http"
	"github.com/hackclub/format/internal/imageproc"
	"github.com/hackclub/format/internal/session"
	"github.com/hackclub/format/internal/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Configure logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ctx := context.Background()

	// Load configuration
	cfg := config.Load()
	logger.Info().Msg("starting format.hackclub.com server")

	// Validate required config
	if cfg.SessionSecret == "" {
		logger.Fatal().Msg("SESSION_SECRET is required")
	}
	if len(cfg.SessionSecret) < 32 {
		logger.Fatal().Msgf("SESSION_SECRET must be at least 32 characters, got %d", len(cfg.SessionSecret))
	}
	logger.Info().Msgf("SESSION_SECRET configured (%d chars), APP_BASE_URL: %s", len(cfg.SessionSecret), cfg.AppBaseURL)
	if cfg.GoogleOAuthClientID == "" {
		logger.Fatal().Msg("GOOGLE_OAUTH_CLIENT_ID is required")
	}
	if cfg.GoogleOAuthClientSecret == "" {
		logger.Fatal().Msg("GOOGLE_OAUTH_CLIENT_SECRET is required")
	}
	if cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" {
		logger.Fatal().Msg("R2 credentials are required")
	}

	// Initialize session manager
	sessionManager := session.NewManager(cfg.SessionSecret, cfg.AppBaseURL)

	// Initialize OIDC provider
	redirectURL := fmt.Sprintf("%s/api/auth/callback", cfg.AppBaseURL)
	oidcProvider, err := auth.NewOIDCProvider(ctx, cfg.GoogleOAuthClientID, cfg.GoogleOAuthClientSecret, redirectURL, cfg.AllowedDomains)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize OIDC provider")
	}

	// Initialize R2 storage client
	r2Client, err := storage.NewR2Client(
		ctx,
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		cfg.R2Bucket,
		cfg.R2S3Endpoint,
		cfg.R2PublicBaseURL,
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize R2 client")
	}

	// Initialize image processor
	processor := imageproc.NewProcessor(
		cfg.JPEGQuality,
		cfg.JPEGProgressive,
		cfg.PNGStrip,
	)

	// Initialize asset service
	assetService := assets.NewService(processor, r2Client, logger)

	// Initialize asset handler
	assetHandler := assets.NewHandler(assetService, logger)

	// Initialize HTML transformer (use configured CDN base)
	htmlTransformer := html.NewTransformer(assetService, cfg.R2PublicBaseURL)

	// Initialize HTTP server
	server := httphandler.NewServer(
		cfg,
		logger,
		sessionManager,
		oidcProvider,
		assetHandler,
		htmlTransformer,
	)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        server.Routes(),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in a goroutine
	go func() {
		logger.Info().Str("port", cfg.Port).Msg("server starting")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("server shutting down")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("server forced to shutdown")
	}

	logger.Info().Msg("server exited")
}
