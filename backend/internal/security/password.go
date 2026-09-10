package security

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type PasswordManager struct {
	mu     sync.RWMutex
	hashed []byte
}

func NewPasswordManager(rawPassword string) (*PasswordManager, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &PasswordManager{
		hashed: hashed,
	}, nil
}

func (p *PasswordManager) Verify(password string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.hashed) == 0 {
		return false
	}
	return bcrypt.CompareHashAndPassword(p.hashed, []byte(password)) == nil
}

func (p *PasswordManager) UpdatePassword(newPassword string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hashed = hashed
	return nil
}

