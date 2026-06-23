// Package words provides the bundled English word list and random generators
// used to build typing tests.
package words

import (
	_ "embed"
	"math/rand"
	"strings"
	"time"
)

//go:embed english.txt
var englishRaw string

// list holds the parsed common-English words, loaded once at init.
var list []string

// rng is the package's random source; reseedable via Seed for reproducible
// runs (used by tests and the recorded demo).
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func init() {
	for _, w := range strings.Fields(englishRaw) {
		w = strings.TrimSpace(w)
		if w != "" {
			list = append(list, w)
		}
	}
}

// Seed makes word generation deterministic for the given seed.
func Seed(seed int64) { rng = rand.New(rand.NewSource(seed)) }

// RNG returns the package's random source so callers (e.g. decoration) share
// the same seedable stream as word generation.
func RNG() *rand.Rand { return rng }

// All returns the full bundled word list.
func All() []string { return list }

// Random returns n words chosen uniformly at random (with repetition),
// avoiding the same word twice in a row for readability.
func Random(n int) []string {
	if n <= 0 || len(list) == 0 {
		return nil
	}
	out := make([]string, 0, n)
	prev := -1
	for i := 0; i < n; i++ {
		idx := rng.Intn(len(list))
		if idx == prev && len(list) > 1 {
			idx = (idx + 1) % len(list)
		}
		prev = idx
		out = append(out, list[idx])
	}
	return out
}
