package cmd

import (
	"runtime/debug"
	"testing"

	"github.com/UltimateForm/protonshell/internal/servers"
)

func TestResolveVersion(t *testing.T) {
	withModule := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: v}}
	}
	tests := []struct {
		name   string
		linked string
		info   *debug.BuildInfo
		want   string
	}{
		{"goreleaser tag drops v", "1.2.0", withModule("(devel)"), "v1.2.0"},
		{"linked wins over module", "1.2.0", withModule("v0.9.0"), "v1.2.0"},
		{"go install module version", "", withModule("v1.3.0"), "v1.3.0"},
		{"devel build", "", withModule("(devel)"), "dev"},
		{"no build info", "", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.linked, tt.info); got != tt.want {
				t.Errorf("resolveVersion(%q, …) = %q, want %q", tt.linked, got, tt.want)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got := parseFlags(nil)
		if got.iface != "proton0" {
			t.Errorf("iface = %q, want proton0", got.iface)
		}
		if got.list != servers.DefaultPath() {
			t.Errorf("list = %q, want %q", got.list, servers.DefaultPath())
		}
	})
	t.Run("overrides", func(t *testing.T) {
		got := parseFlags([]string{"-iface", "wg0", "-list", "/tmp/list.json"})
		if want := (options{list: "/tmp/list.json", iface: "wg0"}); got != want {
			t.Errorf("parseFlags = %+v, want %+v", got, want)
		}
	})
}
