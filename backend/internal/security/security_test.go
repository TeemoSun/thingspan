package security

import (
	"testing"
)

func TestPasswordManager(t *testing.T) {
	pm, err := NewPasswordManager("my-secret-password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !pm.Verify("my-secret-password") {
		t.Fatalf("expected password verification to succeed")
	}

	if pm.Verify("wrong-password") {
		t.Fatalf("expected wrong password verification to fail")
	}

	if err := pm.UpdatePassword("new-secret"); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if !pm.Verify("new-secret") {
		t.Fatalf("expected new password to succeed")
	}
	if pm.Verify("my-secret-password") {
		t.Fatalf("expected old password to fail")
	}
}

func TestJWTManager(t *testing.T) {
	jwtm := NewJWTManager("super-secret-key-12345678901234567890", 15, 7)

	accessToken, err := jwtm.CreateAccessToken()
	if err != nil {
		t.Fatalf("failed to create access token: %v", err)
	}

	claims, err := jwtm.DecodeToken(accessToken, "access")
	if err != nil {
		t.Fatalf("failed to decode access token: %v", err)
	}
	if claims["sub"] != "user" {
		t.Errorf("expected sub='user', got %v", claims["sub"])
	}

	// Should fail with wrong type
	if _, err := jwtm.DecodeToken(accessToken, "refresh"); err != ErrWrongType {
		t.Errorf("expected ErrWrongType, got %v", err)
	}

	refreshToken, err := jwtm.CreateRefreshToken()
	if err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}

	refClaims, err := jwtm.DecodeToken(refreshToken, "refresh")
	if err != nil {
		t.Fatalf("failed to decode refresh token: %v", err)
	}
	if refClaims["type"] != "refresh" {
		t.Errorf("expected type='refresh', got %v", refClaims["type"])
	}

	// Test revocation
	jwtm.RevokeRefreshToken(refreshToken)
	if _, err := jwtm.DecodeToken(refreshToken, "refresh"); err != ErrRevokedToken {
		t.Errorf("expected ErrRevokedToken, got %v", err)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	ip := "192.168.1.100"

	// Initially allowed
	if !rl.CheckLimit(ip) {
		t.Fatalf("expected initial check to pass")
	}

	// 5 failures
	for i := 0; i < 5; i++ {
		if !rl.CheckLimit(ip) {
			t.Fatalf("expected attempt %d to pass check before recording failure", i+1)
		}
		rl.RecordFailure(ip)
	}

	// 6th attempt should be blocked
	if rl.CheckLimit(ip) {
		t.Fatalf("expected 6th attempt to be blocked")
	}

	// Different IP should not be blocked
	otherIP := "192.168.1.101"
	if !rl.CheckLimit(otherIP) {
		t.Fatalf("expected different IP to pass check")
	}
}

