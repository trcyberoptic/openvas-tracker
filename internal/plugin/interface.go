// internal/plugin/interface.go
package plugin

import (
	"context"

	"github.com/cyberoptic/vulntrack/internal/scanner"
)

// Plugin defines the interface for VulnTrack scanner plugins.
// Plugins are compiled as Go plugins (.so) and loaded at runtime.
type Plugin interface {
	// Name returns the plugin's unique identifier
	Name() string
	// Version returns the plugin version
	Version() string
	// Scan runs the plugin's custom scan logic
	Scan(ctx context.Context, target string, options map[string]string) (*scanner.NmapResult, error)
}
