package service

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// EnvFileService reads and writes the .env file for config management via UI.
type EnvFileService struct {
	path string
	mu   sync.Mutex
}

func NewEnvFileService(path string) *EnvFileService {
	return &EnvFileService{path: path}
}

// editableEnvKeys is what the Settings UI may write. The secrets that hand over
// the service itself (JWT secret, import key, admin password, DSN) are absent on
// purpose: they are set on the host, never through the API.
var editableEnvKeys = map[string]bool{
	"OT_SERVER_PORT": true, "OT_JWT_EXPIREHOURS": true, "OT_AUTORESOLVE_THRESHOLD": true, "OT_BUGREPORT_URL": true,
	"OT_LDAP_URL": true, "OT_LDAP_BASE_DN": true, "OT_LDAP_BIND_DN": true, "OT_LDAP_BIND_PASSWORD": true,
	"OT_LDAP_GROUP_DN": true, "OT_LDAP_USER_FILTER": true, "OT_LDAP_INSECURE_SKIP_VERIFY": true,
	"OT_GMP_USER": true, "OT_GMP_PASSWORD": true,
}

var (
	ErrEnvKeyNotEditable = errors.New("env key is not editable via the API")
	ErrEnvValueInvalid   = errors.New("env value must not contain line breaks")
)

// validateEnvPair guards the file format: an unknown key or a value with a line
// break would let the caller append arbitrary variables to the EnvironmentFile.
func validateEnvPair(key, value string) error {
	if !editableEnvKeys[key] {
		return fmt.Errorf("%w: %q", ErrEnvKeyNotEditable, key)
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%w: %s", ErrEnvValueInvalid, key)
	}
	return nil
}

// Read returns all key-value pairs from the .env file.
func (s *EnvFileService) Read() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, scanner.Err()
}

// Update sets or updates a key in the .env file. Preserves comments and order.
func (s *EnvFileService) Update(key, value string) error {
	if err := validateEnvPair(key, value); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	lines, err := s.readLines()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return s.writeLines(lines)
}

// UpdateMultiple sets multiple keys at once.
func (s *EnvFileService) UpdateMultiple(pairs map[string]string) error {
	for k, v := range pairs {
		if err := validateEnvPair(k, v); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	lines, err := s.readLines()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	remaining := make(map[string]string)
	for k, v := range pairs {
		remaining[k] = v
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for key, value := range remaining {
			if strings.HasPrefix(trimmed, key+"=") {
				lines[i] = fmt.Sprintf("%s=%s", key, value)
				delete(remaining, key)
				break
			}
		}
	}

	for key, value := range remaining {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return s.writeLines(lines)
}

func (s *EnvFileService) readLines() ([]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(data), "\n"), nil
}

func (s *EnvFileService) writeLines(lines []string) error {
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(s.path, []byte(content), 0600)
}
