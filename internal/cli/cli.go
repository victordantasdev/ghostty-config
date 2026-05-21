package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	ghosttyconfig "ghostty-config"
	"ghostty-config/internal/app"
	"ghostty-config/internal/ghostty"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Execute() error {
	return rootCmd().Execute()
}

func rootCmd() *cobra.Command {
	opts := ghostty.DefaultOptions()

	cmd := &cobra.Command{
		Use:   "ghostty-config",
		Short: "Configure Ghostty shaders and themes with live preview",
		Long: `ghostty-config opens a CLI to configure Ghostty.

From the main menu you can choose:

  - Shaders: pick custom GLSL shaders (global pipeline + cursor effect)
    and watch the preview live in the current terminal.
  - Themes:  pick a light and a dark theme from your user themes
    directory and Ghostty's bundled themes, with live preview.

Every change is written to the Ghostty config and a reload is triggered
so the running terminal updates immediately.`,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", version, commit, date),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigPath, "config", "c", opts.ConfigPath, "path to Ghostty config")
	cmd.Flags().StringVarP(&opts.ShaderDir, "shader-dir", "s", opts.ShaderDir, "directory containing .glsl shaders")
	cmd.Flags().StringVar(&opts.UserThemeDir, "user-theme-dir", opts.UserThemeDir, "user themes directory")
	cmd.Flags().StringVar(&opts.SystemThemeDir, "system-theme-dir", opts.SystemThemeDir, "Ghostty bundled themes directory")
	cmd.Flags().BoolVar(&opts.NoReload, "no-reload", false, "write the config but do not trigger Ghostty reload")
	cmd.Flags().StringVar(&opts.ReloadCommand, "reload-command", "", "shell command to run after each change instead of the default macOS reload keystroke")

	return cmd
}

func run(opts ghostty.Options) error {
	opts.ConfigPath, _ = filepath.Abs(ghostty.ExpandHome(opts.ConfigPath))
	opts.ShaderDir, _ = filepath.Abs(ghostty.ExpandHome(opts.ShaderDir))
	opts.UserThemeDir, _ = filepath.Abs(ghostty.ExpandHome(opts.UserThemeDir))
	opts.SystemThemeDir, _ = filepath.Abs(ghostty.ExpandHome(opts.SystemThemeDir))

	if err := extractBundledIfMissing(opts.ShaderDir, "shaders"); err != nil {
		return err
	}
	if err := extractBundledIfMissing(opts.UserThemeDir, "themes"); err != nil {
		return err
	}

	root := app.New(opts, version)
	_, err := tea.NewProgram(root, tea.WithAltScreen()).Run()
	return err
}

func extractBundledIfMissing(destDir, embedRoot string) error {
	return extractFSIfMissing(ghosttyconfig.Bundled, destDir, embedRoot)
}

type readFileFS interface {
	fs.FS
	ReadFile(name string) ([]byte, error)
}

func extractFSIfMissing(src readFileFS, destDir, embedRoot string) error {
	if _, err := os.Stat(destDir); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	return fs.WalkDir(src, embedRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(embedRoot, path)
		if rel == "." {
			return nil
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := src.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
