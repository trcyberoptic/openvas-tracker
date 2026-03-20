// internal/auth/password_test.go
package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("secureP@ss1")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "secureP@ss1" {
		t.Fatal("hash must not equal plaintext")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("secureP@ss1")

	if !CheckPassword("secureP@ss1", hash) {
		t.Error("expected password to match hash")
	}
	if CheckPassword("wrongpassword", hash) {
		t.Error("expected wrong password to not match")
	}
}
