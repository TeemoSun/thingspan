package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrWrongType    = errors.New("wrong token type")
	ErrRevokedToken = errors.New("token revoked")
)

type JWTManager struct {
	secret         []byte
	accessDuration time.Duration
	refreshDuration time.Duration

	mu      sync.RWMutex
	revoked map[string]time.Time // jti -> exp
}

func NewJWTManager(secret string, accessMinutes, refreshDays int) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		accessDuration:  time.Duration(accessMinutes) * time.Minute,
		refreshDuration: time.Duration(refreshDays) * 24 * time.Hour,
		revoked:         make(map[string]time.Time),
	}
}

func generateJTI() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (m *JWTManager) createToken(tokenType string, duration time.Duration) (string, error) {
	now := time.Now().UTC()
	jti, err := generateJTI()
	if err != nil {
		return "", fmt.Errorf("failed to generate JTI: %w", err)
	}

	claims := jwt.MapClaims{
		"sub":  "user",
		"type": tokenType,
		"iat":  now.Unix(),
		"exp":  now.Add(duration).Unix(),
		"jti":  jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *JWTManager) CreateAccessToken() (string, error) {
	return m.createToken("access", m.accessDuration)
}

func (m *JWTManager) CreateRefreshToken() (string, error) {
	return m.createToken("refresh", m.refreshDuration)
}

func (m *JWTManager) DecodeToken(tokenString string, expectedType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != expectedType {
		return nil, ErrWrongType
	}

	if expectedType == "refresh" {
		m.cleanupRevoked()
		jti, ok := claims["jti"].(string)
		if ok && jti != "" {
			m.mu.RLock()
			_, isRevoked := m.revoked[jti]
			m.mu.RUnlock()
			if isRevoked {
				return nil, ErrRevokedToken
			}
		}
	}

	return claims, nil
}

func (m *JWTManager) RevokeRefreshToken(tokenString string) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return
	}
	if claims["type"] != "refresh" {
		return
	}
	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return
	}
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return
	}
	exp := time.Unix(int64(expFloat), 0).UTC()

	m.mu.Lock()
	defer m.mu.Unlock()
	m.revoked[jti] = exp
}

func (m *JWTManager) cleanupRevoked() {
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for jti, exp := range m.revoked {
		if !exp.After(now) {
			delete(m.revoked, jti)
		}
	}
}

