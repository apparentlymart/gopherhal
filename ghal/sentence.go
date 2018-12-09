package ghal

import (
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"
	"gopkg.in/jdkato/prose.v2"
)

type Word struct {
	Tag  string
	Text string
}

func MakeWord(tag, text string) Word {
	text = strings.ToLower(norm.NFC.String(text))
	return Word{tag, text}
}

func (w Word) GoString() string {
	return fmt.Sprintf("ghal.MakeWord(%q, %q)", w.Tag, w.Text)
}

type Sentence []Word

func (s Sentence) String() string {
	var ret strings.Builder
	for i, w := range s {
		if i > 0 {
			// We'll probably want to insert a space, but there are some
			// exceptions.
			prev := s[i-1]
			switch {
			case w.Tag == "." || w.Tag == "," || w.Tag == ":" || w.Tag == ")" || w.Tag == "''":
			case prev.Tag == "(" || prev.Tag == "``" || prev.Tag == "$":
			case strings.Contains(w.Text, "'"):
			default:
				// In all other cases we insert a space.
				ret.WriteByte(' ')
			}
		}
		ret.WriteString(w.Text)
	}
	return ret.String()
}

type WordSet map[Word]struct{}

func (s WordSet) Has(k Word) bool {
	_, ok := s[k]
	return ok
}

func (s WordSet) Add(k Word) {
	s[k] = struct{}{}
}

// ChooseRandom will choose up to n words pseudo-randomly from the receiving
// set, returning a slice with n or fewer elements.
func (s WordSet) ChooseRandom(n int) []Word {
	ret := make([]Word, n)
	return s.ChooseRandomInto(ret)
}

// ChooseOneRandom is like ChooseRandom but returns only a single chain.
// Will panic if called on an empty set.
func (s WordSet) ChooseOneRandom() Word {
	for w := range s {
		return w
	}
	panic("ChooseOneRandom on empty WordSet")
}

// ChooseRandomInto is like ChooseRandom but allows the caller to provide the
// target buffer. The length of the given slice decides the maximum number
// to choose, and the result is a slice with the same backing array that may
// be shorter if there were not enough items in the set to fill it.
func (s WordSet) ChooseRandomInto(into []Word) []Word {
	n := len(into)
	into = into[:0]
	i := 0
	// This is relying on the pseudo-random traversal of maps by the
	// Go runtime, which isn't actually guaranteed by Go spec and so this
	// may become more or less random in future versions of Go.
	// Since this package is just a toy, we don't care too much.
	for w := range s {
		into = append(into, w)
		i++
		if i >= n {
			break
		}
	}
	return into
}

func ParseText(text string) ([]Sentence, error) {
	whole, err := prose.NewDocument(text)
	if err != nil {
		return nil, err
	}
	sents := whole.Sentences()
	sentences := make([]Sentence, 0, len(sents))
	for _, s := range sents {
		sDoc, err := prose.NewDocument(s.Text)
		if err != nil {
			return nil, err
		}
		toks := sDoc.Tokens()
		sentence := make(Sentence, len(toks))
		for i, token := range toks {
			sentence[i] = MakeWord(token.Tag, token.Text)
		}
		sentences = append(sentences, sentence)
	}
	return sentences, nil
}
