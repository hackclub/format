package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hackclub/format/internal/assets"
	"github.com/hackclub/format/internal/auth"
	"github.com/hackclub/format/internal/config"
	"github.com/hackclub/format/internal/html"
	"github.com/hackclub/format/internal/session"
	"github.com/rs/zerolog"
)

type Server struct {
	config         *config.Config
	logger         zerolog.Logger
	sessionManager *session.Manager
	oidcProvider   *auth.OIDCProvider
	assetHandler   *assets.Handler
	htmlTransformer *html.Transformer
}

func NewServer(
	cfg *config.Config,
	logger zerolog.Logger,
	sessionManager *session.Manager,
	oidcProvider *auth.OIDCProvider,
	assetHandler *assets.Handler,
	htmlTransformer *html.Transformer,
) *Server {
	return &Server{
		config:         cfg,
		logger:         logger,
		sessionManager: sessionManager,
		oidcProvider:   oidcProvider,
		assetHandler:   assetHandler,
		htmlTransformer: htmlTransformer,
	}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.LoggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.config.AppBaseURL, "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/healthz", s.HealthCheck)

	// Public config endpoint (no auth required)
	r.Get("/api/config", s.HandleConfig)
	
	// Authentication routes (no auth required)
	r.Route("/api/auth", func(r chi.Router) {
		r.Get("/login", s.HandleLogin)
		r.Get("/callback", s.HandleCallback)
		r.Post("/logout", s.HandleLogout)
		r.With(s.AuthMiddleware).Get("/me", s.HandleMe)

	})

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(s.AuthMiddleware)

		// Assets
		r.Post("/assets", s.assetHandler.HandleUpload)
		r.Post("/assets/batch", s.assetHandler.HandleBatch)
		r.Get("/assets/{id}", s.assetHandler.HandleGetAsset)

		// HTML transformation
		r.Post("/html/transform", s.HandleHTMLTransform)

		
	})

	return r
}

// Middleware

func (s *Server) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		
		s.logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", time.Since(start)).
			Str("ip", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Msg("request")
	})
}

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.sessionManager.GetUser(r)
		if err != nil || user == nil {
			s.logger.Debug().Err(err).Msg("authentication failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Handlers

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

func (s *Server) HandleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"cdnBaseUrl": s.config.R2PublicBaseURL,
	})
}


func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state := auth.GenerateState()
	authURL := s.oidcProvider.GetAuthURL(state)

	// Store state in session for validation
	// For simplicity, we'll redirect immediately
	// In production, you might want to store state and validate it in callback

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (s *Server) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		s.logger.Error().Msg("no authorization code received")
		http.Error(w, "Authorization failed", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := s.oidcProvider.ExchangeCode(ctx, code)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to exchange code for token")
		http.Error(w, "Authorization failed", http.StatusInternalServerError)
		return
	}

	// Verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		s.logger.Error().Msg("no id_token in response")
		http.Error(w, "Authorization failed", http.StatusInternalServerError)
		return
	}

	claims, err := s.oidcProvider.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to verify ID token")
		http.Error(w, "Authorization failed - domain not allowed or invalid token", http.StatusForbidden)
		return
	}

	// Create user session
	user := &session.User{
		Sub:     claims.Sub,
		Email:   claims.Email,
		Name:    claims.Name,
		Picture: claims.Picture,
		HD:      claims.HD,
	}

	err = s.sessionManager.SetUser(w, r, user)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to set user session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	s.logger.Info().Str("email", user.Email).Str("domain", user.HD).Msg("user logged in")

	// Create user session (essential for authentication)
	err = s.sessionManager.SetUser(w, r, user)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to set user session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Also pass OAuth tokens to frontend via URL fragment for Gmail API access
	expiresIn := int64(3600) // Default fallback
	if !token.Expiry.IsZero() {
		expiresIn = int64(time.Until(token.Expiry).Seconds())
		if expiresIn <= 0 {
			expiresIn = 0 // Token already expired
		}
	}
	
	redirectURL := fmt.Sprintf("%s#access_token=%s&expires_in=%d", 
		s.config.AppBaseURL, 
		token.AccessToken,
		expiresIn)
	
	if token.RefreshToken != "" {
		redirectURL += "&refresh_token=" + token.RefreshToken
	}
	
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	err := s.sessionManager.ClearSession(w, r)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to clear session")
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "logged out"})
}

func (s *Server) HandleMe(w http.ResponseWriter, r *http.Request) {
	userValue := r.Context().Value("user")
	if userValue == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	
	user, ok := userValue.(*session.User)
	if !ok {
		http.Error(w, "Invalid user context", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) HandleHTMLTransform(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req html.TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.HTML == "" {
		http.Error(w, "HTML content required", http.StatusBadRequest)
		return
	}

	result, err := s.htmlTransformer.Transform(ctx, &req)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to transform HTML")
		http.Error(w, "Failed to transform HTML", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}


