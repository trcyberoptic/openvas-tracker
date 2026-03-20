package database

import (
	"testing"
)

func TestNewPool_InvalidURL(t *testing.T) {
	_, err := NewPool("postgres://invalid:5432/nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid database URL, got nil")
	}
}

func TestNewPool_ValidatesConfig(t *testing.T) {
	_, err := NewPool("")
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}
