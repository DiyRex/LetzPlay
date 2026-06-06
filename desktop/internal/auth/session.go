package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
)

const cookieName = "LETZPLAY_SESSION"

// Session is the authenticated principal stored in the signed cookie.
type Session struct {
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
}

// SessionManager signs and verifies session cookies with an HMAC secret. The secret is generated
// per launch, so cookies are invalidated on restart — fine for a single-evening party server.
type SessionManager struct {
	secret []byte
}

// NewSessionManager wraps a signing secret (see crypto/rand in main).
func NewSessionManager(secret []byte) *SessionManager {
	return &SessionManager{secret: secret}
}

// Write sets the signed session cookie on the response.
func (m *SessionManager) Write(w http.ResponseWriter, s Session) {
	payload, _ := json.Marshal(s)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	value := encoded + "." + m.sign(encoded)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear expires the session cookie.
func (m *SessionManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
}

// Read returns the verified session from the request, or an error if absent/tampered.
func (m *SessionManager) Read(r *http.Request) (Session, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, errors.New("no session")
	}
	encoded, sig, ok := splitDot(cookie.Value)
	if !ok || subtle.ConstantTimeCompare([]byte(sig), []byte(m.sign(encoded))) != 1 {
		return Session{}, errors.New("invalid session signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(payload, &s); err != nil {
		return Session{}, err
	}
	return s, nil
}

func (m *SessionManager) sign(value string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func splitDot(s string) (value, sig string, ok bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}
