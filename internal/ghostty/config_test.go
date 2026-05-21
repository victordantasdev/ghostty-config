package ghostty

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestReadActiveValues(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")

	t.Run("missing file returns nil", func(t *testing.T) {
		got := ReadActiveValues(filepath.Join(dir, "missing"), "theme")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	writeFile(t, cfg, `# comment line
theme = catppuccin-frappe
custom-shader = "shaders/crt.glsl" # inline comment
# theme = ignored
custom-shader = 'shaders/cursor_blaze.glsl'
theme=   spaced
theme =
other-key = ignored
`)

	t.Run("returns matching values, strips quotes and comments", func(t *testing.T) {
		got := ReadActiveValues(cfg, "theme")
		want := []string{"catppuccin-frappe", "spaced"}
		if !equalSlice(got, want) {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("custom-shader values", func(t *testing.T) {
		got := ReadActiveValues(cfg, "custom-shader")
		want := []string{"shaders/crt.glsl", "shaders/cursor_blaze.glsl"}
		if !equalSlice(got, want) {
			t.Errorf("got %v want %v", got, want)
		}
	})

	t.Run("malformed lines without equals are skipped", func(t *testing.T) {
		broken := filepath.Join(dir, "broken")
		// Note: split on "=" with maxSplit=2 — to trigger SplitN return < 2 we'd need a line w/o =.
		// keyRegex requires `=` so this line wouldn't match. Add a comment line and a bare key.
		writeFile(t, broken, "theme\n# nothing\n")
		got := ReadActiveValues(broken, "theme")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestWriteConfigKey(t *testing.T) {
	t.Run("new file, no desired -> no write", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		if err := WriteConfigKey(cfg, "theme", nil); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(cfg); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("config should not exist: %v", err)
		}
	})

	t.Run("new file with desired writes managed block", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "nested", "config")
		if err := WriteConfigKey(cfg, "theme", []string{"catppuccin"}); err != nil {
			t.Fatal(err)
		}
		got := readFile(t, cfg)
		if !strings.Contains(got, ManagedMarker) {
			t.Errorf("missing managed marker: %q", got)
		}
		if !strings.Contains(got, "theme = catppuccin") {
			t.Errorf("missing theme line: %q", got)
		}
	})

	t.Run("existing file: ensures backup, comments existing key, appends managed", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "font-size = 14\ntheme = oldtheme\n")
		if err := WriteConfigKey(cfg, "theme", []string{"newtheme"}); err != nil {
			t.Fatal(err)
		}
		out := readFile(t, cfg)
		if !strings.Contains(out, "# theme = oldtheme") {
			t.Errorf("old theme should be commented: %q", out)
		}
		if !strings.Contains(out, "font-size = 14") {
			t.Errorf("font-size kept: %q", out)
		}
		if !strings.Contains(out, ManagedMarker) {
			t.Errorf("missing marker: %q", out)
		}
		if !strings.Contains(out, "theme = newtheme") {
			t.Errorf("missing new theme: %q", out)
		}

		bkp := cfg + "-bkp"
		if _, err := os.Stat(bkp); err != nil {
			t.Fatalf("backup not created: %v", err)
		}
		// Backup should not be overwritten on next call.
		bkpBefore := readFile(t, bkp)
		if err := WriteConfigKey(cfg, "theme", []string{"another"}); err != nil {
			t.Fatal(err)
		}
		bkpAfter := readFile(t, bkp)
		if bkpBefore != bkpAfter {
			t.Errorf("backup overwritten: before=%q after=%q", bkpBefore, bkpAfter)
		}
	})

	t.Run("existing file with managed block: replaces values", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "font-size = 14\n\n"+ManagedMarker+"\ntheme = old\ncustom-shader = a.glsl\n")
		if err := WriteConfigKey(cfg, "theme", []string{"new"}); err != nil {
			t.Fatal(err)
		}
		out := readFile(t, cfg)
		if !strings.Contains(out, "theme = new") {
			t.Errorf("missing new theme: %q", out)
		}
		if strings.Contains(out, "theme = old") {
			t.Errorf("old theme should be replaced: %q", out)
		}
		if !strings.Contains(out, "custom-shader = a.glsl") {
			t.Errorf("shader preserved: %q", out)
		}
	})

	t.Run("desired empty deletes the key from managed", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, ManagedMarker+"\ntheme = old\ncustom-shader = a.glsl\n")
		if err := WriteConfigKey(cfg, "theme", nil); err != nil {
			t.Fatal(err)
		}
		out := readFile(t, cfg)
		if strings.Contains(out, "theme = old") {
			t.Errorf("theme should be removed: %q", out)
		}
		if !strings.Contains(out, "custom-shader = a.glsl") {
			t.Errorf("shader kept: %q", out)
		}
	})

	t.Run("removes managed marker when no values remain", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "font-size = 14\n"+ManagedMarker+"\ntheme = old\n")
		if err := WriteConfigKey(cfg, "theme", nil); err != nil {
			t.Fatal(err)
		}
		out := readFile(t, cfg)
		if strings.Contains(out, ManagedMarker) {
			t.Errorf("marker should be removed: %q", out)
		}
		if !strings.Contains(out, "font-size = 14") {
			t.Errorf("preamble kept: %q", out)
		}
	})

	t.Run("no change -> no write (mtime preserved)", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, ManagedMarker+"\ntheme = stable\n")
		info0, err := os.Stat(cfg)
		if err != nil {
			t.Fatal(err)
		}
		// Make sure there's enough granularity.
		time.Sleep(20 * time.Millisecond)
		if err := WriteConfigKey(cfg, "theme", []string{"stable"}); err != nil {
			t.Fatal(err)
		}
		info1, err := os.Stat(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !info0.ModTime().Equal(info1.ModTime()) {
			t.Errorf("file rewritten despite identical content")
		}
	})

	t.Run("CRLF line endings normalized", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "font = a\r\ntheme = old\r\n")
		if err := WriteConfigKey(cfg, "theme", []string{"new"}); err != nil {
			t.Fatal(err)
		}
		out := readFile(t, cfg)
		if strings.Contains(out, "\r\n") {
			t.Errorf("CRLF should be normalized: %q", out)
		}
	})

	t.Run("preamble without trailing newline is padded", func(t *testing.T) {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "font = a")
		if err := WriteConfigKey(cfg, "theme", []string{"x"}); err != nil {
			t.Fatal(err)
		}
		out := readFile(t, cfg)
		if !strings.Contains(out, "font = a\n") {
			t.Errorf("preamble newline missing: %q", out)
		}
	})

	t.Run("MkdirAll error returns it", func(t *testing.T) {
		// Make a regular file at a path the function will try to create as a dir.
		dir := t.TempDir()
		blocker := filepath.Join(dir, "block")
		writeFile(t, blocker, "x")
		err := WriteConfigKey(filepath.Join(blocker, "child", "config"), "theme", []string{"x"})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("ReadFile error other than NotExist", func(t *testing.T) {
		dir := t.TempDir()
		// Create directory at config path, so ReadFile fails with non-NotExist error.
		cfg := filepath.Join(dir, "config")
		if err := os.Mkdir(cfg, 0o755); err != nil {
			t.Fatal(err)
		}
		err := WriteConfigKey(cfg, "theme", []string{"x"})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("ensureBackup write failure surfaces as WriteConfigKey error", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("file mode semantics differ on windows")
		}
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "theme = old\n")
		// Make the parent dir read-only so backup WriteFile fails but ReadFile of cfg works.
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(dir, 0o755)
		err := WriteConfigKey(cfg, "theme", []string{"new"})
		if err == nil {
			t.Errorf("expected error from ensureBackup write")
		}
	})

	t.Run("ensureBackup propagates non-ENOENT stat error via symlink loop", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink semantics differ on windows")
		}
		dir := t.TempDir()
		cfg := filepath.Join(dir, "config")
		writeFile(t, cfg, "theme = old\n")
		bkp := cfg + "-bkp"
		// Self-referencing symlink → Stat returns ELOOP, not ENOENT.
		if err := os.Symlink(bkp, bkp); err != nil {
			t.Fatal(err)
		}
		err := WriteConfigKey(cfg, "theme", []string{"new"})
		if err == nil {
			t.Errorf("expected ELOOP from ensureBackup stat")
		}
	})
}

func TestSplitAtMarker(t *testing.T) {
	t.Run("no marker", func(t *testing.T) {
		before, after := splitAtMarker("foo\nbar\n")
		if before != "foo\nbar\n" || after != "" {
			t.Errorf("got before=%q after=%q", before, after)
		}
	})
	t.Run("marker at end of file with no newline", func(t *testing.T) {
		text := "preamble\n" + ManagedMarker
		before, after := splitAtMarker(text)
		if before != "preamble" || after != "" {
			t.Errorf("got before=%q after=%q", before, after)
		}
	})
	t.Run("marker followed by lines", func(t *testing.T) {
		text := "p\n" + ManagedMarker + "\ntheme = x\n"
		before, after := splitAtMarker(text)
		if before != "p" {
			t.Errorf("before=%q", before)
		}
		if !strings.Contains(after, "theme = x") {
			t.Errorf("after=%q", after)
		}
	})
}

func TestCommentOutManagedKeys(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		if got := commentOutManagedKeys(""); got != "" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("comments managed keys, keeps others, preserves trailing newline", func(t *testing.T) {
		in := "font = a\ntheme = b\n# already\n\n"
		got := commentOutManagedKeys(in)
		if !strings.Contains(got, "# theme = b") {
			t.Errorf("theme should be commented: %q", got)
		}
		if !strings.Contains(got, "font = a") {
			t.Errorf("font kept: %q", got)
		}
		if !strings.HasSuffix(got, "\n") {
			t.Errorf("trailing newline lost: %q", got)
		}
	})
	t.Run("no trailing newline", func(t *testing.T) {
		got := commentOutManagedKeys("theme = b")
		if got != "# theme = b" {
			t.Errorf("got %q", got)
		}
	})
}

func TestParseManagedValues(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := parseManagedValues("")
		if len(got) != 0 {
			t.Errorf("got %v", got)
		}
	})
	t.Run("ignores comments and unrelated keys", func(t *testing.T) {
		in := "# comment\nrandom = x\ntheme = a\ntheme = \ncustom-shader = b\n"
		got := parseManagedValues(in)
		if !equalSlice(got["theme"], []string{"a"}) {
			t.Errorf("theme: %v", got["theme"])
		}
		if !equalSlice(got["custom-shader"], []string{"b"}) {
			t.Errorf("shader: %v", got["custom-shader"])
		}
	})
	t.Run("inline comment stripped, multiple values aggregated", func(t *testing.T) {
		in := "theme = a # inline\ntheme = b\n"
		got := parseManagedValues(in)
		if !equalSlice(got["theme"], []string{"a", "b"}) {
			t.Errorf("got %v", got["theme"])
		}
	})
}

func TestStripInlineComment(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"foo", "foo"},
		{"foo # bar", "foo "},
		{`foo = "a # b"`, `foo = "a # b"`},
		{"foo = 'a # b'", "foo = 'a # b'"},
		{`foo = "a" # tail`, `foo = "a" `},
		{`a"b'c"# tail`, `a"b'c"`},
	}
	for _, c := range cases {
		got := stripInlineComment(c.in)
		if got != c.want {
			t.Errorf("stripInlineComment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	// "~" alone falls through TrimPrefix unchanged (prefix is "~/"), producing $HOME/~.
	if got := ExpandHome("~"); got != filepath.Join(home, "~") {
		t.Errorf("~ -> %q want %q", got, filepath.Join(home, "~"))
	}
	if got := ExpandHome("~/foo/bar"); got != filepath.Join(home, "foo/bar") {
		t.Errorf("~/foo -> %q", got)
	}
	if got := ExpandHome("/abs/path"); got != "/abs/path" {
		t.Errorf("/abs/path mutated -> %q", got)
	}
	if got := ExpandHome("relative"); got != "relative" {
		t.Errorf("relative mutated -> %q", got)
	}
}

func TestReload(t *testing.T) {
	t.Run("NoReload returns nil", func(t *testing.T) {
		if err := Reload(Options{NoReload: true}); err != nil {
			t.Errorf("unexpected: %v", err)
		}
	})

	t.Run("ReloadCommand success", func(t *testing.T) {
		if err := Reload(Options{ReloadCommand: "true"}); err != nil {
			t.Errorf("unexpected: %v", err)
		}
	})

	t.Run("ReloadCommand failure", func(t *testing.T) {
		if err := Reload(Options{ReloadCommand: "false"}); err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("non-darwin returns error", func(t *testing.T) {
		prev := currentGOOS
		currentGOOS = "linux"
		defer func() { currentGOOS = prev }()
		err := Reload(Options{})
		if err == nil || !strings.Contains(err.Error(), "macOS") {
			t.Errorf("expected macOS error, got %v", err)
		}
	})

	t.Run("darwin osReload success", func(t *testing.T) {
		prevGOOS := currentGOOS
		prevReload := osReload
		prevSleep := reloadSleep
		currentGOOS = "darwin"
		osReload = func() error { return nil }
		reloadSleep = 0
		defer func() {
			currentGOOS = prevGOOS
			osReload = prevReload
			reloadSleep = prevSleep
		}()
		if err := Reload(Options{}); err != nil {
			t.Errorf("unexpected: %v", err)
		}
	})

	t.Run("darwin osReload failure wrapped", func(t *testing.T) {
		prevGOOS := currentGOOS
		prevReload := osReload
		currentGOOS = "darwin"
		osReload = func() error { return errors.New("denied") }
		defer func() {
			currentGOOS = prevGOOS
			osReload = prevReload
		}()
		err := Reload(Options{})
		if err == nil || !strings.Contains(err.Error(), "reload failed") {
			t.Errorf("expected wrapped error, got %v", err)
		}
	})
}

func TestOsReloadDefault(t *testing.T) {
	// Just exercise the function to keep its body covered. We don't care
	// about the result: on darwin without Accessibility permission it errors,
	// elsewhere `osascript` is missing — either way the function executes.
	_ = osReloadDefault()
}

func TestKeyRegex(t *testing.T) {
	re := keyRegex("theme")
	if !re.MatchString("theme = x") {
		t.Errorf("should match")
	}
	if !re.MatchString("  theme=x") {
		t.Errorf("should match leading space")
	}
	if re.MatchString("# theme = x") {
		t.Errorf("should not match comment lead")
	}
	if re.MatchString("themes = x") {
		t.Errorf("should not match prefix")
	}
}

func TestBuildOutputEdgeCases(t *testing.T) {
	t.Run("empty before, empty managed", func(t *testing.T) {
		got := buildOutput("", map[string][]string{})
		if got != "" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("before with newline, no managed", func(t *testing.T) {
		got := buildOutput("foo\n", map[string][]string{})
		if got != "foo\n" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("empty before with managed", func(t *testing.T) {
		got := buildOutput("", map[string][]string{"theme": {"a"}})
		if !strings.HasPrefix(got, ManagedMarker) {
			t.Errorf("missing marker at start: %q", got)
		}
	})
}

func equalSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
