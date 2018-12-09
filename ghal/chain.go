package ghal

import (
	"fmt"
	"math/rand"
)

const chainLen = 4

type chain [chainLen]Word

func makeChain(words []Word) chain {
	if len(words) != chainLen {
		panic("incorrect number of words for chain")
	}
	var ret chain
	for i := range ret {
		ret[i] = words[i]
	}
	return ret
}

func (c *chain) GoString() string {
	return fmt.Sprintf("ghal.makeChain(%#v)", c[:])
}

// PushBefore modifies the receiver in-place so that the first three words
// are shifted along one position, the final word is lost, and the given
// new word is placed in the first position.
func (c *chain) PushBefore(word Word) {
	c[0], c[1], c[2], c[3] = word, c[0], c[1], c[2]
}

// PushAfter modifies the receiver in-place so that the last three words
// are shifted back one position, the first word is lost, and the given
// new word is placed in the last position.
func (c *chain) PushAfter(word Word) {
	c[0], c[1], c[2], c[3] = c[1], c[2], c[3], word
}

type chainSet map[chain]struct{}

func (s chainSet) Has(c chain) bool {
	_, ok := s[c]
	return ok
}

func (s chainSet) Add(c chain) {
	s[c] = struct{}{}
}

// ChooseRandom will choose up to n chains pseudo-randomly from the receiving
// set, returning a slice with n or fewer elements.
func (s chainSet) ChooseRandom(n int) []chain {
	ret := make([]chain, n)
	return s.ChooseRandomInto(ret)
}

// ChooseOneRandom is like ChooseRandom but returns only a single chain.
// Will panic if called on an empty set.
func (s chainSet) ChooseOneRandom() chain {
	if len(s) == 0 {
		panic("ChooseOneRandom on empty chainSet")
	}
	ofs := rand.Int() % len(s)
	i := 0
	for c := range s {
		if i == ofs {
			return c
		}
		i++
	}
	panic("ChooseOneRandom on empty chainSet")
}

// ChooseRandomInto is like ChooseRandom but allows the caller to provide the
// target buffer. The length of the given slice decides the maximum number
// to choose, and the result is a slice with the same backing array that may
// be shorter if there were not enough items in the set to fill it.
func (s chainSet) ChooseRandomInto(into []chain) []chain {
	n := len(into)
	into = into[:0]
	i := 0
	// This is relying on the pseudo-random traversal of maps by the
	// Go runtime, which isn't actually guaranteed by Go spec and so this
	// may become more or less random in future versions of Go.
	// Since this package is just a toy, we don't care too much.
	for c := range s {
		into = append(into, c)
		i++
		if i >= n {
			break
		}
	}
	return into
}
