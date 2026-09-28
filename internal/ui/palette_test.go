package ui

import (
	"fmt"
	"testing"

	"charm.land/lipgloss/v2"
)

// The tidemark palette is validated per variant; make sure each background gets its own steps
// and that ansi switches back to terminal colours.
func TestPaletteVariants(t *testing.T) {
	defer setPalette("ansi", true)
	hex := func() string {
		r, g, b, _ := cCPU.GetForeground().RGBA()
		return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	setPalette("tidemark", true)
	if got := hex(); got != "#3987e5" {
		t.Errorf("dark cpu = %s, want #3987e5", got)
	}
	setPalette("tidemark", false)
	if got := hex(); got != "#2a78d6" {
		t.Errorf("light cpu = %s, want #2a78d6", got)
	}
	setPalette("ansi", false)
	if got := cCPU.GetForeground(); got != lipgloss.ANSIColor(6) {
		t.Errorf("ansi cpu = %v, want ANSI slot 6", got)
	}
}
