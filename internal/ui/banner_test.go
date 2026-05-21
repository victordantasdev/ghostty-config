package ui

import (
	"strings"
	"testing"
)

func TestRenderWarnBanner(t *testing.T) {
	got := RenderWarnBanner("hello")
	if !strings.Contains(got, "⚠") {
		t.Errorf("missing warn icon: %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("missing text: %q", got)
	}
}

func TestRenderToast(t *testing.T) {
	cases := []struct {
		kind     ToastKind
		prefix   string
		expected string
	}{
		{ToastSaved, "✓", "saved text"},
		{ToastReverted, "↩", "reverted text"},
		{ToastInfo, "", "info text"},
	}
	for _, c := range cases {
		got := RenderToast(c.expected, c.kind)
		if !strings.Contains(got, c.expected) {
			t.Errorf("kind=%d missing text: %q", c.kind, got)
		}
		if c.prefix != "" && !strings.Contains(got, c.prefix) {
			t.Errorf("kind=%d missing prefix %q: %q", c.kind, c.prefix, got)
		}
	}
}

func TestRenderBreadcrumb(t *testing.T) {
	t.Run("empty returns empty", func(t *testing.T) {
		if got := RenderBreadcrumb(); got != "" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("single segment", func(t *testing.T) {
		got := RenderBreadcrumb("Root")
		if !strings.Contains(got, "Root") {
			t.Errorf("missing Root: %q", got)
		}
		if strings.Contains(got, "›") {
			t.Errorf("unexpected separator with one segment: %q", got)
		}
	})
	t.Run("multiple segments includes separators", func(t *testing.T) {
		got := RenderBreadcrumb("A", "B", "C")
		for _, want := range []string{"A", "B", "C", "›"} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in %q", want, got)
			}
		}
	})
}

func TestScreenAndToastKindConstants(t *testing.T) {
	// Sanity: distinct values.
	if ScreenMenu == ScreenShader || ScreenShader == ScreenTheme || ScreenMenu == ScreenTheme {
		t.Errorf("Screen constants collide")
	}
	if ToastSaved == ToastReverted || ToastReverted == ToastInfo {
		t.Errorf("ToastKind constants collide")
	}
}
