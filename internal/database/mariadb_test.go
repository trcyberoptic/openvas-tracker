package database

import "testing"

func TestNewPool_InvalidDSN(t *testing.T) {
	_, err := NewPool(PoolConfig{DSN: "invalid:invalid@tcp(localhost:9999)/nonexistent?timeout=1s"})
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

func TestNewPool_ValidatesConfig(t *testing.T) {
	_, err := NewPool(PoolConfig{})
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}
