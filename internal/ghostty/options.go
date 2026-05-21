package ghostty

import (
	"os"
	"path/filepath"
)

type Options struct {
	ConfigPath     string
	ShaderDir      string
	UserThemeDir   string
	SystemThemeDir string
	ReloadCommand  string
	NoReload       bool
}

func DefaultOptions() Options {
	home, _ := os.UserHomeDir()
	return Options{
		ConfigPath:     filepath.Join(home, ".config", "ghostty", "config"),
		ShaderDir:      filepath.Join(home, ".config", "ghostty", "shaders"),
		UserThemeDir:   filepath.Join(home, ".config", "ghostty", "themes"),
		SystemThemeDir: "/Applications/Ghostty.app/Contents/Resources/ghostty/themes",
	}
}
