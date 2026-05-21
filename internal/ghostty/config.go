package ghostty

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const ManagedMarker = "# The following settings are managed by ghostty-config. Do not edit by hand."

var managedKeys = []string{"theme", "custom-shader"}

func keyRegex(key string) *regexp.Regexp {
	return regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*=`)
}

func ReadActiveValues(configPath, key string) []string {
	f, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	re := keyRegex(key)
	var values []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := stripInlineComment(s.Text())
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || !re.MatchString(trimmed) {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}

func WriteConfigKey(configPath, key string, desired []string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(configPath)
	fileExists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if !fileExists {
		if len(desired) == 0 {
			return nil
		}
		var b strings.Builder
		b.WriteString(ManagedMarker)
		b.WriteByte('\n')
		writeKeyValues(&b, key, desired)
		return os.WriteFile(configPath, []byte(b.String()), 0o644)
	}

	if err := ensureBackup(configPath, data); err != nil {
		return err
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	before, after := splitAtMarker(text)

	before = commentOutManagedKeys(before)
	managed := parseManagedValues(after)

	if len(desired) == 0 {
		delete(managed, key)
	} else {
		managed[key] = desired
	}

	out := buildOutput(before, managed)
	if out == text {
		return nil
	}
	return os.WriteFile(configPath, []byte(out), 0o644)
}

func ensureBackup(configPath string, data []byte) error {
	backupPath := configPath + "-bkp"
	if _, err := os.Stat(backupPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(backupPath, data, 0o644)
}

func splitAtMarker(text string) (before, after string) {
	marker := strings.TrimSpace(ManagedMarker)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == marker {
			before = strings.Join(lines[:i], "\n")
			if i+1 < len(lines) {
				after = strings.Join(lines[i+1:], "\n")
			}
			return before, after
		}
	}
	return text, ""
}

func commentOutManagedKeys(s string) string {
	if s == "" {
		return s
	}
	hadTrailingNewline := strings.HasSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	if hadTrailingNewline && len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, key := range managedKeys {
			if keyRegex(key).MatchString(trimmed) {
				lines[i] = "# " + line
				break
			}
		}
	}

	out := strings.Join(lines, "\n")
	if hadTrailingNewline {
		out += "\n"
	}
	return out
}

func parseManagedValues(s string) map[string][]string {
	out := map[string][]string{}
	if s == "" {
		return out
	}
	for _, line := range strings.Split(s, "\n") {
		stripped := stripInlineComment(line)
		trimmed := strings.TrimSpace(stripped)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, key := range managedKeys {
			if keyRegex(key).MatchString(trimmed) {
				parts := strings.SplitN(trimmed, "=", 2)
				if len(parts) == 2 {
					value := strings.TrimSpace(parts[1])
					if value != "" {
						out[key] = append(out[key], value)
					}
				}
				break
			}
		}
	}
	return out
}

func buildOutput(before string, managed map[string][]string) string {
	var b strings.Builder
	b.WriteString(before)
	if before != "" && !strings.HasSuffix(before, "\n") {
		b.WriteByte('\n')
	}

	hasManaged := false
	for _, k := range managedKeys {
		if len(managed[k]) > 0 {
			hasManaged = true
			break
		}
	}
	if !hasManaged {
		return b.String()
	}

	if before != "" {
		b.WriteByte('\n')
	}
	b.WriteString(ManagedMarker)
	b.WriteByte('\n')
	for _, k := range managedKeys {
		writeKeyValues(&b, k, managed[k])
	}
	return b.String()
}

func writeKeyValues(b *strings.Builder, key string, values []string) {
	for _, v := range values {
		b.WriteString(key)
		b.WriteString(" = ")
		b.WriteString(v)
		b.WriteByte('\n')
	}
}

func Reload(opts Options) error {
	if opts.NoReload {
		return nil
	}

	if opts.ReloadCommand != "" {
		cmd := exec.Command("/bin/sh", "-c", opts.ReloadCommand)
		return cmd.Run()
	}

	if runtime.GOOS != "darwin" {
		return fmt.Errorf("automatic reload is only built in for macOS; pass --reload-command or --no-reload")
	}

	cmd := exec.Command("osascript", "-e", `tell application "System Events" to keystroke "," using {command down, shift down}`)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reload failed (grant Ghostty/terminal Accessibility permission, use --reload-command, or --no-reload): %w", err)
	}

	time.Sleep(35 * time.Millisecond)
	return nil
}

func stripInlineComment(line string) string {
	inSingle := false
	inDouble := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return line[:i]
			}
		}
	}
	return line
}

func ExpandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
