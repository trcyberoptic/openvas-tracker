package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvFile_RejectsKeyOutsideAllowlist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	s := NewEnvFileService(path)
	for _, key := range []string{
		"OT_JWT_SECRET", "OT_IMPORT_APIKEY", "OT_ADMIN_PASSWORD", "OT_DATABASE_DSN", // service-controlling secrets
		"PATH", "OT_LDAP_URL\nOT_ADMIN_PASSWORD", "", // foreign / injected keys
	} {
		if err := s.Update(key, "x"); !errors.Is(err, ErrEnvKeyNotEditable) {
			t.Errorf("Update(%q): err = %v, want ErrEnvKeyNotEditable", key, err)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("env file was written despite rejected key")
	}
}

func TestEnvFile_RejectsLineBreaksInValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	s := NewEnvFileService(path)
	for _, v := range []string{"ldap://x\nOT_ADMIN_PASSWORD=y", "ldap://x\r\nOT_JWT_SECRET=z"} {
		if err := s.Update("OT_LDAP_URL", v); !errors.Is(err, ErrEnvValueInvalid) {
			t.Errorf("Update(value %q): err = %v, want ErrEnvValueInvalid", v, err)
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("env file was written despite rejected value")
	}
}

func TestEnvFile_UpdateMultipleIsAllOrNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	s := NewEnvFileService(path)
	err := s.UpdateMultiple(map[string]string{"OT_LDAP_URL": "ldaps://dc01", "OT_JWT_SECRET": "x"})
	if !errors.Is(err, ErrEnvKeyNotEditable) {
		t.Fatalf("err = %v, want ErrEnvKeyNotEditable", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("env file was written although one key was rejected")
	}
}

func TestEnvFile_UpdateWritesAllowedKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	s := NewEnvFileService(path)
	if err := s.Update("OT_LDAP_URL", "ldaps://dc01.example.com:636"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "OT_LDAP_URL=ldaps://dc01.example.com:636\n") {
		t.Errorf("file = %q", data)
	}
}
