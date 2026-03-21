package service

import (
	"bufio"
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
