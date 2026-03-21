package service

import (
	"fmt"
	"testing"
)

func TestMapSeverity(t *testing.T) {
	tests := []struct {
		threat string
		cvss   float64
		want   string
	}{
		{"High", 9.5, "critical"},
		{"High", 7.5, "high"},
		{"Medium", 5.0, "medium"},
		{"Low", 2.0, "low"},
		{"Log", 0.0, "info"},
		{"Debug", 0.0, "info"},
		{"", 0.0, "info"},
	}
	for _, tt := range tests {
		got := mapSeverity(tt.threat, tt.cvss)
		if got != tt.want {
			t.Errorf("mapSeverity(%q, %.1f) = %q, want %q", tt.threat, tt.cvss, got, tt.want)
		}
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		input     string
		wantPort  *int32
		wantProto *string
	}{
		{"443/tcp", int32Ptr(443), strPtrTest("tcp")},
		{"80/tcp", int32Ptr(80), strPtrTest("tcp")},
		{"general/tcp", nil, nil},
		{"", nil, nil},
	}
	for _, tt := range tests {
		port, proto := parsePort(tt.input)
		if !int32PtrEq(port, tt.wantPort) || !strPtrEq(proto, tt.wantProto) {
			t.Errorf("parsePort(%q) = (%v, %v), want (%v, %v)", tt.input, derefInt32(port), derefStr(proto), derefInt32(tt.wantPort), derefStr(tt.wantProto))
		}
	}
}

func int32Ptr(v int32) *int32        { return &v }
func strPtrTest(v string) *string    { return &v }
func derefInt32(p *int32) string     { if p == nil { return "<nil>" }; return fmt.Sprintf("%d", *p) }
func derefStr(p *string) string      { if p == nil { return "<nil>" }; return *p }
func int32PtrEq(a, b *int32) bool    { if a == nil && b == nil { return true }; if a == nil || b == nil { return false }; return *a == *b }
func strPtrEq(a, b *string) bool     { if a == nil && b == nil { return true }; if a == nil || b == nil { return false }; return *a == *b }
