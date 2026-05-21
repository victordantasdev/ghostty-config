package ghosttyconfig

import (
	"io/fs"
	"strings"
	"testing"
)

func TestBundledContainsShadersAndThemes(t *testing.T) {
	for _, root := range []string{"shaders", "themes"} {
		entries, err := fs.ReadDir(Bundled, root)
		if err != nil {
			t.Errorf("ReadDir %q: %v", root, err)
			continue
		}
		if len(entries) == 0 {
			t.Errorf("embed root %q is empty", root)
		}
	}
}

func TestBundledContainsGlsl(t *testing.T) {
	got := 0
	_ = fs.WalkDir(Bundled, "shaders", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".glsl") {
			got++
		}
		return nil
	})
	if got == 0 {
		t.Errorf("expected at least one .glsl shader bundled")
	}
}
