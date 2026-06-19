// Package stats computes typing-test metrics: WPM, raw WPM, accuracy,
// consistency, and the per-second sample series used for the results chart.
package stats

import "math"

// Sample is one per-second data point captured during a test.
type Sample struct {
	T      float64 // elapsed seconds at capture
	WPM    float64 // net words-per-minute at this point
	Raw    float64 // raw words-per-minute (all keystrokes count)
	Errors int     // incorrect characters committed during this second
}

// Result is the final, immutable summary of a completed test.
type Result struct {
	WPM         float64
	Raw         float64
	Accuracy    float64 // 0..100
	Consistency float64 // 0..100
	Seconds     float64

	Correct    int // correctly typed characters
	Incorrect  int // wrong characters
	Extra      int // characters typed past a word's length
	Missed     int // characters in words skipped via space
	Keystrokes int

	Samples []Sample
}

// WPM returns net words-per-minute: (correct chars / 5) per minute.
func WPM(correctChars int, seconds float64) float64 {
	if seconds <= 0 {
		return 0
	}
	return (float64(correctChars) / 5.0) / (seconds / 60.0)
}

// Raw returns raw words-per-minute using every typed character.
func Raw(typedChars int, seconds float64) float64 {
	if seconds <= 0 {
		return 0
	}
	return (float64(typedChars) / 5.0) / (seconds / 60.0)
}

// Accuracy returns the percentage of correct keystrokes (0..100).
func Accuracy(correct, keystrokes int) float64 {
	if keystrokes <= 0 {
		return 100
	}
	return 100.0 * float64(correct) / float64(keystrokes)
}

// Consistency derives a 0..100 score from the coefficient of variation of
// the per-second raw WPM series. Steadier typing scores higher.
func Consistency(samples []Sample) float64 {
	if len(samples) < 2 {
		return 100
	}
	var sum float64
	for _, s := range samples {
		sum += s.Raw
	}
	mean := sum / float64(len(samples))
	if mean == 0 {
		return 0
	}
	var variance float64
	for _, s := range samples {
		d := s.Raw - mean
		variance += d * d
	}
	variance /= float64(len(samples))
	cv := math.Sqrt(variance) / mean
	c := 100.0 * (1.0 - cv)
	if c < 0 {
		c = 0
	}
	if c > 100 {
		c = 100
	}
	return c
}
