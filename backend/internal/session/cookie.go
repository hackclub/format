package session

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "format-session"
	UserKey     = "user"

	oauthStateKey        = "oauth_state"
	oauthCodeVerifierKey = "oauth_code_verifier"
)

type Manager struct {
	store sessions.Store
}

type User struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	HD      string `json:"hd"`
}

type TokenInfo struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

// NewManager configures cookie flags based on APP_BASE_URL
func NewManager(sessionSecret string, appBaseURL string) *Manager {
	store := sessions.NewCookieStore([]byte(sessionSecret))

	secure := false
	sameSite := http.SameSiteLaxMode // recommended for OAuth code flow
	if u, err := url.Parse(appBaseURL); err == nil && strings.EqualFold(u.Scheme, "https") {
		secure = true
	}

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   12 * 60 * 60, // 12 hours
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	}

	return &Manager{store: store}
}

func (m *Manager) SetUser(w http.ResponseWriter, r *http.Request, user *User) error {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return err
	}
	userBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}
	sess.Values[UserKey] = string(userBytes)
	return sess.Save(r, w)
}

func (m *Manager) GetUser(r *http.Request) (*User, error) {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return nil, err
	}
	userStr, ok := sess.Values[UserKey].(string)
	if !ok || userStr == "" {
		return nil, nil
	}
	var user User
	if err := json.Unmarshal([]byte(userStr), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *Manager) ClearSession(w http.ResponseWriter, r *http.Request) error {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return err
	}
	sess.Values[UserKey] = ""
	sess.Values[oauthStateKey] = ""
	sess.Values[oauthCodeVerifierKey] = ""
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}

func (m *Manager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := m.GetUser(r)
		if err != nil || user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- OAuth helpers ---

func (m *Manager) SetOAuthState(w http.ResponseWriter, r *http.Request, state string) error {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		// If the session is corrupted, create a new one
		sess, err = m.store.New(r, SessionName)
		if err != nil {
			return err
		}
	}
	sess.Values[oauthStateKey] = state
	return sess.Save(r, w)
}

func (m *Manager) GetAndClearOAuthState(w http.ResponseWriter, r *http.Request) (string, error) {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return "", err
	}
	state, _ := sess.Values[oauthStateKey].(string)
	sess.Values[oauthStateKey] = ""
	if err := sess.Save(r, w); err != nil {
		return "", err
	}
	return state, nil
}

func (m *Manager) SetOAuthCodeVerifier(w http.ResponseWriter, r *http.Request, verifier string) error {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return err
	}
	sess.Values[oauthCodeVerifierKey] = verifier
	return sess.Save(r, w)
}

func (m *Manager) GetAndClearOAuthCodeVerifier(w http.ResponseWriter, r *http.Request) (string, error) {
	sess, err := m.store.Get(r, SessionName)
	if err != nil {
		return "", err
	}
	verifier, _ := sess.Values[oauthCodeVerifierKey].(string)
	sess.Values[oauthCodeVerifierKey] = ""
	if err := sess.Save(r, w); err != nil {
		return "", err
	}
	return verifier, nil
}
