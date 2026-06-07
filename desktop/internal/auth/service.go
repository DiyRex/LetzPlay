// Package auth handles password verification, role resolution, and signed session cookies for
// the desktop server. It mirrors the Android app's model: there are no per-user accounts — the
// password presented selects the role (admin vs guest).
package auth

import (
	"sync"

	"github.com/DiyRex/LetzPlay/desktop/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// Service verifies passwords against bcrypt hashes. Hashes can be updated at runtime (admin panel)
// under a lock, so reads in Authenticate stay race-free.
type Service struct {
	mu            sync.RWMutex
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
	s.mu.RLock()
	adminHash, guestHash, guestRequired := s.adminHash, s.guestHash, s.guestRequired
	s.mu.RUnlock()

	if bcrypt.CompareHashAndPassword(adminHash, []byte(password)) == nil {
		return domain.RoleAdmin, true
	}
	if bcrypt.CompareHashAndPassword(guestHash, []byte(password)) == nil {
		return domain.RoleGuest, true
	}
	if !guestRequired {
		return domain.RoleGuest, true
	}
	return "", false
}

// SetAdminPassword / SetGuestPassword update credentials at runtime (in-memory; resets on restart).
func (s *Service) SetAdminPassword(plain string) error { return s.set(&s.adminHash, plain) }
func (s *Service) SetGuestPassword(plain string) error { return s.set(&s.guestHash, plain) }

func (s *Service) set(field *[]byte, plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.mu.Lock()
	*field = hash
	s.mu.Unlock()
	return nil
}
