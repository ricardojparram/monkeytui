package words

import (
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

func TestDecorateNoFlagsIsIdentity(t *testing.T) {
	in := []string{"the", "quick", "brown"}
	out := Decorate(in, false, false, rand.New(rand.NewSource(1)))
	if strings.Join(out, " ") != strings.Join(in, " ") {
		t.Fatalf("no-flags should be identity, got %v", out)
	}
}

func TestDecorateEmptySafe(t *testing.T) {
	if out := Decorate(nil, true, true, rand.New(rand.NewSource(1))); len(out) != 0 {
		t.Fatalf("nil input should stay empty, got %v", out)
	}
}

func TestDecorateDeterministicForSeed(t *testing.T) {
	in := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}
	a := Decorate(in, true, true, rand.New(rand.NewSource(42)))
	b := Decorate(in, true, true, rand.New(rand.NewSource(42)))
	if strings.Join(a, " ") != strings.Join(b, " ") {
		t.Fatalf("same seed must produce same output:\n%v\n%v", a, b)
	}
}

func TestDecoratePunctuationCapitalizesFirst(t *testing.T) {
	in := []string{"alpha", "beta", "gamma"}
	out := Decorate(in, true, false, rand.New(rand.NewSource(7)))
	if out[0][0] < 'A' || out[0][0] > 'Z' {
		t.Fatalf("first word must be capitalized, got %q", out[0])
	}
}

func TestDecorateNumbersAreDigitsAndNotAdjacent(t *testing.T) {
	in := make([]string, 40)
	for i := range in {
		in[i] = "word"
	}
	out := Decorate(in, false, true, rand.New(rand.NewSource(3)))
	numRe := regexp.MustCompile(`^[0-9]{1,4}$`)
	prevNum := false
	sawNum := false
	for _, w := range out {
		isNum := numRe.MatchString(w)
		if isNum {
			sawNum = true
			if prevNum {
				t.Fatalf("two number tokens adjacent in %v", out)
			}
		}
		prevNum = isNum
	}
	if !sawNum {
		t.Fatalf("expected at least one number token in %v", out)
	}
}
