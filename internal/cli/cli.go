package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	ghosttyconfig "ghostty-config"
	"ghostty-config/internal/app"
	"ghostty-config/internal/ghostty"
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
	var err error
	opts.ConfigPath, err = filepath.Abs(ghostty.ExpandHome(opts.ConfigPath))
	if err != nil {
		return err
	}
	opts.ShaderDir, err = filepath.Abs(ghostty.ExpandHome(opts.ShaderDir))
	if err != nil {
		return err
	}
	opts.UserThemeDir, err = filepath.Abs(ghostty.ExpandHome(opts.UserThemeDir))
	if err != nil {
		return err
	}
	opts.SystemThemeDir, err = filepath.Abs(ghostty.ExpandHome(opts.SystemThemeDir))
	if err != nil {
		return err
	}

	if err := extractBundledIfMissing(opts.ShaderDir, "shaders"); err != nil {
		return err
	}
	if err := extractBundledIfMissing(opts.UserThemeDir, "themes"); err != nil {
		return err
	}

	root := app.New(opts)
	_, err = tea.NewProgram(root, tea.WithAltScreen()).Run()
	return err
}

func extractBundledIfMissing(destDir, embedRoot string) error {
	if _, err := os.Stat(destDir); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	return fs.WalkDir(ghosttyconfig.Bundled, embedRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(embedRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := ghosttyconfig.Bundled.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
