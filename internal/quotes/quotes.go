// Package quotes provides a small bundled set of quotes for quote-mode tests.
package quotes

import "math/rand"

// Quote is a single bundled quote with attribution.
type Quote struct {
	Text   string
	Source string
}

var all = []Quote{
	{"The only way to do great work is to love what you do.", "Steve Jobs"},
	{"In the middle of difficulty lies opportunity.", "Albert Einstein"},
	{"Whether you think you can or you think you can't, you're right.", "Henry Ford"},
	{"The best way to predict the future is to invent it.", "Alan Kay"},
	{"Simplicity is the ultimate sophistication.", "Leonardo da Vinci"},
	{"It does not matter how slowly you go as long as you do not stop.", "Confucius"},
	{"Premature optimization is the root of all evil.", "Donald Knuth"},
	{"Talk is cheap. Show me the code.", "Linus Torvalds"},
	{"Programs must be written for people to read, and only incidentally for machines to execute.", "Harold Abelson"},
	{"The function of good software is to make the complex appear to be simple.", "Grady Booch"},
	{"First, solve the problem. Then, write the code.", "John Johnson"},
	{"Code is like humor. When you have to explain it, it is bad.", "Cory House"},
	{"Make it work, make it right, make it fast.", "Kent Beck"},
	{"The most damaging phrase in the language is we have always done it this way.", "Grace Hopper"},
	{"Any fool can write code that a computer can understand. Good programmers write code that humans can understand.", "Martin Fowler"},
}

// Random returns a random bundled quote.
func Random() Quote {
	return all[rand.Intn(len(all))]
}

// All returns every bundled quote.
func All() []Quote { return all }
