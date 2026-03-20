// internal/plugin/loader.go
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	goplugin "plugin"
)

type Registry struct {
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func (r *Registry) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no plugins dir is fine
		}
		return err
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".so" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := r.Load(path); err != nil {
			return fmt.Errorf("failed to load plugin %s: %w", path, err)
		}
	}
	return nil
}

func (r *Registry) Load(path string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return err
	}
	sym, err := p.Lookup("OpenVASTrackerPlugin")
	if err != nil {
		return fmt.Errorf("plugin missing OpenVASTrackerPlugin symbol: %w", err)
	}
	plug, ok := sym.(Plugin)
	if !ok {
		return fmt.Errorf("OpenVASTrackerPlugin does not implement Plugin interface")
	}
	r.plugins[plug.Name()] = plug
	return nil
}

func (r *Registry) Get(name string) (Plugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
