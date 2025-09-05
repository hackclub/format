package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OIDCProvider struct {
	config         *oauth2.Config
	verifier       *oidc.IDTokenVerifier
	allowedDomains map[string]bool
	firstDomain    string // used for Google hd hint
}

type Claims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	HD            string `json:"hd"` // hosted domain
}

func NewOIDCProvider(ctx context.Context, clientID, clientSecret, redirectURL string, allowedDomains []string) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     google.Endpoint,
		Scopes: []string{
			oidc.ScopeOpenID, "profile", "email",
			"https://www.googleapis.com/auth/gmail.readonly",
		},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	domainMap := make(map[string]bool)
	for _, d := range allowedDomains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			domainMap[d] = true
		}
	}
	// choose a stable first domain for hd hint
	var keys []string
	for k := range domainMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	first := ""
	if len(keys) > 0 {
		first = keys[0]
	}

	return &OIDCProvider{
		config:         config,
		verifier:       verifier,
		allowedDomains: domainMap,
		firstDomain:    first,
	}, nil
}

func (p *OIDCProvider) GetAuthURL(state, codeChallenge string) string {
	params := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("access_type", "offline"),             // allow refresh tokens (server-side)
		oauth2.SetAuthURLParam("prompt", "consent"),                  // consistent scope grant
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),      // PKCE
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),      // PKCE
	}
	if p.firstDomain != "" {
		params = append(params, oauth2.SetAuthURLParam("hd", p.firstDomain)) // hint only
	}
	return p.config.AuthCodeURL(state, params...)
}

func (p *OIDCProvider) VerifyIDToken(ctx context.Context, idToken string) (*Claims, error) {
	token, err := p.verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims Claims
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	if !claims.EmailVerified {
		return nil, fmt.Errorf("email not verified")
	}

	if claims.HD == "" {
		return nil, fmt.Errorf("no hosted domain found in token - personal accounts not allowed")
	}

	if !p.allowedDomains[strings.ToLower(claims.HD)] {
		return nil, fmt.Errorf("domain %s is not allowed", claims.HD)
	}

	return &claims, nil
}

func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string, codeVerifier string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
}

func GenerateState() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func GeneratePKCEVerifier() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func PKCEChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
