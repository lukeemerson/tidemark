package ui

import "charm.land/lipgloss/v2"

// Palettes: "ansi" (default) draws with the terminal's 16 colours, so its theme applies.
// "tidemark" is a fixed truecolor set, one variant stepped for dark backgrounds and one for
// light, checked with the dataviz validator against #1a1a19, #1d1d1d and #282828 (dark) and
// #fcfcfb and #fbf1c7 (light). Tile order cpu·gpu·power·mem·temp·↓·↑ passes lightness, chroma
// and the normal-vision floor (ΔE ≥ 19.7) in both. Known limits, all covered by visible labels:
// dark ↓/↑ (green/yellow) is ΔE 6.9 for protan vision, and on light backgrounds aqua, magenta
// and yellow sit under 3:1 contrast.
type swatch struct{ light, dark string }

var tidemarkPalette = struct {
	cpu, gpu, power, mem, temp, down, up, muted swatch
	low, mid, high                              swatch
}{
	cpu:   swatch{"#2a78d6", "#3987e5"}, // blue
	gpu:   swatch{"#1baf7a", "#199e70"}, // aqua
	power: swatch{"#eb6834", "#d95926"}, // orange
	mem:   swatch{"#4a3aa7", "#9085e9"}, // violet
	temp:  swatch{"#e87ba4", "#d55181"}, // magenta
	down:  swatch{"#008300", "#008300"}, // green
	up:    swatch{"#eda100", "#c98500"}, // yellow
	muted: swatch{"#898781", "#898781"}, // frames, labels, ping
	low:   swatch{"#0ca30c", "#0ca30c"}, // state: good
	mid:   swatch{"#fab219", "#fab219"}, // state: warning
	high:  swatch{"#d03b3b", "#d03b3b"}, // state: critical
}

func ansiFg(n int) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(n)) }

// setPalette points every colour role at the named palette ("" = ansi); dark picks the tidemark
// variant for the terminal's background.
func setPalette(name string, dark bool) {
	if name != "tidemark" {
		// slots chosen so neighbouring tiles differ even where bright slots equal normal ones
		dim, low, mid, high = ansiFg(8), ansiFg(2), ansiFg(3), ansiFg(1)
		cCPU, cGPU, cPower, cMem = ansiFg(6), ansiFg(4), ansiFg(5), ansiFg(14)
		// upload/write: slot 7 on dark backgrounds, slot 0 on light ones, where 7 can be pale grey
		up := 7
		if !dark {
			up = 0
		}
		cTemp, cDown, cUp, cPing = ansiFg(13), ansiFg(12), ansiFg(up), ansiFg(8)
		return
	}
	p := tidemarkPalette
	pick := func(s swatch) lipgloss.Style {
		c := s.light
		if dark {
			c = s.dark
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(c))
	}
	dim, low, mid, high = pick(p.muted), pick(p.low), pick(p.mid), pick(p.high)
	cCPU, cGPU, cPower, cMem = pick(p.cpu), pick(p.gpu), pick(p.power), pick(p.mem)
	cTemp, cDown, cUp, cPing = pick(p.temp), pick(p.down), pick(p.up), pick(p.muted)
}
