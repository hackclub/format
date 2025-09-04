package session

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionName = "format-session"
	UserKey     = "user"
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

func NewManager(sessionSecret string) *Manager {
	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   12 * 60 * 60, // 12 hours
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}
	
	return &Manager{
		store: store,
	}
}

func (m *Manager) SetUser(w http.ResponseWriter, r *http.Request, user *User) error {
	session, err := m.store.Get(r, SessionName)
	if err != nil {
		return err
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}

	session.Values[UserKey] = string(userBytes)
	return session.Save(r, w)
}

func (m *Manager) GetUser(r *http.Request) (*User, error) {
	session, err := m.store.Get(r, SessionName)
	if err != nil {
		return nil, err
	}

	userStr, ok := session.Values[UserKey].(string)
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
	session, err := m.store.Get(r, SessionName)
	if err != nil {
		return err
	}

	session.Values[UserKey] = ""
	session.Options.MaxAge = -1
	return session.Save(r, w)
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
