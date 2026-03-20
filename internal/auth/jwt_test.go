// internal/auth/jwt_test.go
package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()
	role := "admin"

	token, err := GenerateToken(userID, role, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken error: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}

	claims, err := ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Role != role {
		t.Errorf("expected role %s, got %s", role, claims.Role)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret-key"
	userID := uuid.New()

	token, _ := GenerateToken(userID, "viewer", secret, -1*time.Hour)
	_, err := ValidateToken(token, secret)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := GenerateToken(uuid.New(), "viewer", "secret1", 1*time.Hour)
	_, err := ValidateToken(token, "secret2")
	if err == nil {
		t.Fatal("expected error for wrong secret")
	}
}
