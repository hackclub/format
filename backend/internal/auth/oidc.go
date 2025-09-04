package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OIDCProvider struct {
	config       *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	allowedDomains map[string]bool
}

type Claims struct {
	Email    string `json:"email"`
	Sub      string `json:"sub"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	HD       string `json:"hd"` // Hosted domain claim for G Suite/Workspace
}

func NewOIDCProvider(ctx context.Context, clientID, clientSecret, redirectURL string, allowedDomains []string) (*OIDCProvider, error) {
	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %v", err)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     google.Endpoint,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "https://www.googleapis.com/auth/gmail.readonly"},
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	// Convert allowed domains to map for faster lookup
	domainMap := make(map[string]bool)
	for _, domain := range allowedDomains {
		domainMap[strings.TrimSpace(domain)] = true
	}

	return &OIDCProvider{
		config:       config,
		verifier:     verifier,
		allowedDomains: domainMap,
	}, nil
}

func (p *OIDCProvider) GetAuthURL(state string) string {
	scopes := p.config.Scopes
	fmt.Printf("🔍 OAuth scopes being requested: %v\n", scopes)
	
	authURL := p.config.AuthCodeURL(state, 
		oauth2.SetAuthURLParam("hd", strings.Join(p.getAllowedDomains(), " ")))
		
	fmt.Printf("🔗 OAuth URL: %s\n", authURL)
	return authURL
}

func (p *OIDCProvider) VerifyIDToken(ctx context.Context, idToken string) (*Claims, error) {
	token, err := p.verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %v", err)
	}

	var claims Claims
	if err := token.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %v", err)
	}

	// Check if the hosted domain is allowed
	if claims.HD == "" {
		return nil, fmt.Errorf("no hosted domain found in token - personal accounts not allowed")
	}

	if !p.allowedDomains[claims.HD] {
		return nil, fmt.Errorf("domain %s is not allowed", claims.HD)
	}

	return &claims, nil
}

func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *OIDCProvider) getAllowedDomains() []string {
	domains := make([]string, 0, len(p.allowedDomains))
	for domain := range p.allowedDomains {
		domains = append(domains, domain)
	}
	return domains
}

func GenerateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
