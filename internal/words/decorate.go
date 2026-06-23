package words

import (
	"math/rand"
	"strings"
)

// sentenceEnders are trailing marks that start a new sentence (next word capitalized).
const sentenceEnders = ".?!"

// trailingMarks is the weighted pool of punctuation appended to a word: commas
// and periods are most common, then sentence enders and clause marks.
var trailingMarks = []string{".", ".", ".", ",", ",", ",", "?", "!", ";", ":"}

// wraps pairs an opening and closing decoration applied around a bare word.
var wraps = [][2]string{{`"`, `"`}, {"(", ")"}, {"[", "]"}, {"-", "-"}}

// Decorate optionally rewrites generated words with monkeytype-style
// punctuation and/or numbers. With both flags false it returns in unchanged.
// rng drives all randomness so callers can make output reproducible.
func Decorate(in []string, punctuation, numbers bool, rng *rand.Rand) []string {
	if len(in) == 0 || (!punctuation && !numbers) {
		return in
	}
	out := make([]string, len(in))
	prevNum := false
	capNext := true // first word starts a "sentence"
	for i, w := range in {
		token := w
		isNum := false

		// Numbers: replace ~15% of words, never two in a row.
		if numbers && !prevNum && rng.Intn(100) < 15 {
			n := 1 + rng.Intn(4) // 1..4 digits
			b := make([]byte, n)
			for j := range b {
				b[j] = byte('0' + rng.Intn(10))
			}
			token = string(b)
			isNum = true
		}
		prevNum = isNum

		if punctuation && !isNum {
			// Capitalize sentence starts plus ~10% of other words.
			if capNext || rng.Intn(100) < 10 {
				token = capitalize(token)
			}
			capNext = false
			// ~3% wrapped.
			if rng.Intn(100) < 3 {
				wr := wraps[rng.Intn(len(wraps))]
				token = wr[0] + token + wr[1]
			}
			// ~15% trailing mark.
			if rng.Intn(100) < 15 {
				mark := trailingMarks[rng.Intn(len(trailingMarks))]
				token += mark
				if strings.ContainsAny(mark, sentenceEnders) {
					capNext = true
				}
			}
		}
		out[i] = token
	}
	return out
}

// capitalize upper-cases the first rune of s.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}
