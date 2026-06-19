package ui

import (
	"math"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/linechart"
	"github.com/charmbracelet/lipgloss"
	"github.com/ricardojparram/monkeytui/internal/stats"
	"github.com/ricardojparram/monkeytui/internal/theme"
)

// renderChart draws a braille line chart (via ntcharts) of WPM in the accent
// color and raw WPM faint, with red 'x' marks where mistakes happened. width
// and height are the full chart dimensions in cells (axis labels included).
func renderChart(samples []stats.Sample, th theme.Theme, width, height int) string {
	if len(samples) < 2 || width < 20 || height < 5 {
		return th.Faint.Render("(keep typing — not enough data for a graph)")
	}

	maxX := samples[len(samples)-1].T
	if maxX < 1 {
		maxX = 1
	}
	maxY := 0.0
	for _, s := range samples {
		maxY = math.Max(maxY, math.Max(s.WPM, s.Raw))
	}
	maxY = math.Ceil(maxY/20) * 20
	if maxY <= 0 {
		maxY = 20
	}

	lc := linechart.New(width, height, 0, maxX, 0, maxY,
		linechart.WithStyles(th.Faint, th.Faint, lipgloss.NewStyle()),
		linechart.WithXYSteps(5, 4),
	)
	lc.DrawXYAxisAndLabel()

	drawSeries := func(get func(stats.Sample) float64, style lipgloss.Style) {
		for i := 1; i < len(samples); i++ {
			p1 := canvas.Float64Point{X: samples[i-1].T, Y: get(samples[i-1])}
			p2 := canvas.Float64Point{X: samples[i].T, Y: get(samples[i])}
			lc.DrawBrailleLineWithStyle(p1, p2, style)
		}
	}
	drawSeries(func(s stats.Sample) float64 { return s.Raw }, th.Faint)
	drawSeries(func(s stats.Sample) float64 { return s.WPM }, th.Main)

	// Mistakes: like monkeytype, each errored second is a red dot placed on its
	// own "errors" scale — height = how many errors happened that second, scaled
	// to the worst second. So 1-error and 2-error seconds sit at different
	// heights rather than all in a row.
	maxErr := 0
	for _, s := range samples {
		if s.Errors > maxErr {
			maxErr = s.Errors
		}
	}
	dot := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	if maxErr > 0 {
		for _, s := range samples {
			if s.Errors > 0 {
				// Map error count onto the upper portion of the plot so dots
				// never collide with the y=0 axis.
				frac := float64(s.Errors) / float64(maxErr)
				y := maxY * (0.45 + 0.5*frac)
				lc.DrawRuneWithStyle(canvas.Float64Point{X: s.T, Y: y}, '•', dot)
			}
		}
	}

	legend := th.Main.Render("● wpm") + th.Faint.Render("   · raw   ") + dot.Render("• errors")
	return lipgloss.JoinVertical(lipgloss.Left, lc.View(), legend)
}
