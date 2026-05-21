package ghostty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	home, _ := os.UserHomeDir()
	wantConfig := filepath.Join(home, ".config", "ghostty", "config")
	if opts.ConfigPath != wantConfig {
		t.Errorf("ConfigPath = %q, want %q", opts.ConfigPath, wantConfig)
	}
	if !strings.HasSuffix(opts.ShaderDir, filepath.Join(".config", "ghostty", "shaders")) {
		t.Errorf("ShaderDir unexpected: %q", opts.ShaderDir)
	}
	if !strings.HasSuffix(opts.UserThemeDir, filepath.Join(".config", "ghostty", "themes")) {
		t.Errorf("UserThemeDir unexpected: %q", opts.UserThemeDir)
	}
	if opts.SystemThemeDir != "/Applications/Ghostty.app/Contents/Resources/ghostty/themes" {
		t.Errorf("SystemThemeDir = %q", opts.SystemThemeDir)
	}
	if opts.NoReload {
		t.Errorf("NoReload should default to false")
	}
	if opts.ReloadCommand != "" {
		t.Errorf("ReloadCommand should default empty")
	}
}
