package ui

import "strings"

func RenderWarnBanner(text string) string {
	return WarnBannerStyle.Render("⚠ " + text)
}

func RenderToast(text string, kind ToastKind) string {
	switch kind {
	case ToastSaved:
		return ToastSavedStyle.Render("✓ " + text)
	case ToastReverted:
		return ToastRevertedStyle.Render("↩ " + text)
	default:
		return FooterLabelStyle.Render(text)
	}
}

func RenderBreadcrumb(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	parts := make([]string, 0, len(segments)*2-1)
	for i, s := range segments {
		if i == 0 {
			parts = append(parts, BreadcrumbHeadStyle.Render(s))
			continue
		}
		parts = append(parts, BreadcrumbSepStyle.Render(" › "))
		parts = append(parts, BreadcrumbSegStyle.Render(s))
	}
	return strings.Join(parts, "")
}
