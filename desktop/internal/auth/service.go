// Package auth handles password verification, role resolution, and signed session cookies for
// the desktop server. It mirrors the Android app's model: there are no per-user accounts — the
// password presented selects the role (admin vs guest).
package auth

import (
	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// Service verifies passwords against bcrypt hashes computed once at startup.
type Service struct {
	adminHash     []byte
	guestHash     []byte
	guestRequired bool
}

// NewService hashes the configured passwords. guestRequired=false lets anyone in as a guest.
func NewService(adminPassword, guestPassword string, guestRequired bool) (*Service, error) {
	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	guestHash, err := bcrypt.GenerateFromPassword([]byte(guestPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &Service{adminHash: adminHash, guestHash: guestHash, guestRequired: guestRequired}, nil
}

// Authenticate resolves a role from a password: admin hash wins, then guest hash, then (if guests
// are open) any password yields a guest. Returns ok=false when nothing matches.
func (s *Service) Authenticate(password string) (domain.Role, bool) {
	if bcrypt.CompareHashAndPassword(s.adminHash, []byte(password)) == nil {
		return domain.RoleAdmin, true
	}
	if bcrypt.CompareHashAndPassword(s.guestHash, []byte(password)) == nil {
		return domain.RoleGuest, true
	}
	if !s.guestRequired {
		return domain.RoleGuest, true
	}
	return "", false
}
