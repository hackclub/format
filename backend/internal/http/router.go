package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

// helper to compute allowed origin from APP_BASE_URL
func originFromBaseURL(base string) string {
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "http://localhost:3000"
	}
	return fmt.Sprintf("%s://%s", strings.ToLower(u.Scheme), u.Host)
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.LoggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS: dynamically allow only APP_BASE_URL origin (and localhost during local dev)
	allowed := []string{originFromBaseURL(s.config.AppBaseURL)}
	if strings.Contains(s.config.AppBaseURL, "localhost") {
		if !contains(allowed, "http://localhost:3000") {
			allowed = append(allowed, "http://localhost:3000")
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowed,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/healthz", s.HealthCheck)

	// Serve static files from Next.js build
	r.Handle("/_next/static/*", http.StripPrefix("/_next/static/", http.FileServer(http.Dir("./static"))))
	r.Handle("/favicon.svg", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/favicon.svg")
	}))
	
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
		// Accept sharded keys like ab/xxxxxxxx.jpg
		r.Get("/assets/*", s.assetHandler.HandleGetAsset)

		// HTML transformation
		r.Post("/html/transform", s.HandleHTMLTransform)

		
	})

	// Catch-all for SPA routing - serve index.html for any unmatched routes
	r.NotFound(s.HandleSPA)

	return r
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
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
	// Generate state + PKCE
	state := auth.GenerateState()
	verifier := auth.GeneratePKCEVerifier()
	challenge := auth.PKCEChallengeS256(verifier)

	// Persist in session
	if err := s.sessionManager.SetOAuthState(w, r, state); err != nil {
		s.logger.Error().Err(err).Msg("failed to store oauth state")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if err := s.sessionManager.SetOAuthCodeVerifier(w, r, verifier); err != nil {
		s.logger.Error().Err(err).Msg("failed to store oauth code verifier")
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	authURL := s.oidcProvider.GetAuthURL(state, challenge)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (s *Server) HandleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Validate state
	stateParam := r.URL.Query().Get("state")
	if stateParam == "" {
		http.Error(w, "Missing state", http.StatusBadRequest)
		return
	}
	expectedState, err := s.sessionManager.GetAndClearOAuthState(w, r)
	if err != nil || expectedState == "" || expectedState != stateParam {
		s.logger.Error().Err(err).Msg("invalid oauth state")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	verifier, err := s.sessionManager.GetAndClearOAuthCodeVerifier(w, r)
	if err != nil || verifier == "" {
		s.logger.Error().Err(err).Msg("missing code verifier")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Exchange code
	code := r.URL.Query().Get("code")
	if code == "" {
		s.logger.Error().Msg("no authorization code received")
		http.Error(w, "Authorization failed", http.StatusBadRequest)
		return
	}
	token, err := s.oidcProvider.ExchangeCode(ctx, code, verifier)
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

	// Create user session (essential for authentication)
	err = s.sessionManager.SetUser(w, r, user)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to set user session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	s.logger.Info().Str("email", user.Email).Str("domain", user.HD).Msg("user logged in")

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

func (s *Server) HandleSPA(w http.ResponseWriter, r *http.Request) {
	// For any non-API routes, serve the main HTML page (SPA)
	w.Header().Set("Content-Type", "text/html")
	
	// Next.js App Router HTML shell
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Format - Hack Club</title>
    <link rel="icon" href="/favicon.svg" type="image/svg+xml">
    <meta name="next-head-count" content="4">
</head>
<body>
    <div id="__next"></div>
    <script src="/_next/static/chunks/polyfills-42372ed130431b0a.js" nomodule=""></script>
    <script src="/_next/static/chunks/webpack-ac9a027431ef0133.js"></script>
    <script src="/_next/static/chunks/fd9d1056-e6fad75ea1edeaa8.js"></script>
    <script src="/_next/static/chunks/117-37af661815ca3999.js"></script>
    <script src="/_next/static/chunks/main-app-e0137810acce9719.js"></script>
</body>
</html>`
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (s *Server) HandleHTMLTransform(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// limit HTML size (e.g., 1.5MB)
	r.Body = http.MaxBytesReader(w, r.Body, 1_500_000)

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


