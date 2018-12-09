package ghal

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"golang.org/x/text/unicode/norm"
	prose "gopkg.in/jdkato/prose.v2"
)

type Word struct {
	Tag  string
	Text string
}

var (
	_ json.Marshaler   = (*Word)(nil)
	_ json.Unmarshaler = (*Word)(nil)
)

var Period = MakeWord(".", ".")
var QuestionMark = MakeWord(".", "?")
var ExclamationMark = MakeWord(".", "!")

func MakeWord(tag, text string) Word {
	text = strings.ToLower(norm.NFC.String(text))
	return Word{tag, text}
}

func (w Word) GoString() string {
	return fmt.Sprintf("ghal.MakeWord(%q, %q)", w.Tag, w.Text)
}

func (w Word) IsNoun() bool {
	switch w.Tag {
	case "NN", "NNS", "NNP", "NNPS":
		return true
	default:
		return false
	}
}

func (w Word) IsProperNoun() bool {
	switch w.Tag {
	case "NNP", "NNPS":
		return true
	default:
		return false
	}
}

func (w Word) IsHashtag() bool {
	return w.IsNoun() && len(w.Text) > 0 && w.Text[0] == '#'
}

func (w Word) IsAtMention() bool {
	return w.IsNoun() && len(w.Text) > 0 && w.Text[0] == '@'
}

func (w Word) MarshalJSON() ([]byte, error) {
	return json.Marshal([...]string{w.Text, w.Tag})
}

func (w *Word) UnmarshalJSON(src []byte) error {
	var v [2]string
	err := json.Unmarshal(src, &v)
	w.Text = v[0]
	w.Tag = v[1]
	return err
}

type Sentence []Word

// Words returns a set of all of the distinct words in the sentence.
func (s Sentence) Words() WordSet {
	ret := make(WordSet, len(s))
	for _, w := range s {
		ret.Add(w)
	}
	return ret
}

// Nouns returns a set of all of the distinct nouns in the sentence.
func (s Sentence) Nouns() WordSet {
	ret := make(WordSet, len(s))
	for _, w := range s {
		if w.IsNoun() {
			ret.Add(w)
		}
	}
	return ret
}

// ProperNouns returns a set of all of the distinct proper nouns in the sentence.
func (s Sentence) ProperNouns() WordSet {
	ret := make(WordSet, len(s))
	for _, w := range s {
		if w.IsProperNoun() {
			ret.Add(w)
		}
	}
	return ret
}

// TrimPeriod tests whether the final "word" in the receiver is a period and
// if so returns a new slice with the same backing array that does not include
// that trailing period. Otherwise, returns the receiver verbatim.
//
// This method does not trim any other sort of sentence-terminating punctuation,
// such as question marks. It is intended to emulate common casual chatroom
// writing style where periods are usually elided at the ends of sentences.
//
// This can either be used prior to adding a new sentence to a brain (to
// cause the brain to learn sentences without trailing periods) or on a
// sentence constructed by a brain (to cosmetically remove trailing periods
// even though the brain itself considers them part of a sentence).
func (s Sentence) TrimPeriod() Sentence {
	switch {
	case len(s) == 0:
		return s
	case s[len(s)-1] == Period:
		// As a special case, if the token right before the period is _also_
		// a period then we'll leave things as-is, assuming we've found an
		// ellipsis.
		if len(s) > 1 && s[len(s)-2] == Period {
			return s
		}
		return s[:len(s)-1]
	default:
		return s
	}
}

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

// StringTagged is a variant of String that includes the parts-of-speech tag
// information, using the common word/TAG notation.
func (s Sentence) StringTagged() string {
	var ret strings.Builder
	for i, w := range s {
		if i > 0 {
			ret.WriteByte(' ')
		}
		ret.WriteString(w.Text)
		ret.WriteByte('/')
		ret.WriteString(w.Tag)
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

// Nouns returns the set of words within the receiver that are nouns.
func (s WordSet) Nouns() WordSet {
	ret := make(WordSet)
	for w := range s {
		if w.IsNoun() {
			ret.Add(w)
		}
	}
	return ret
}

// Union returns the union of the receiver and all of the other given sets.
func (s WordSet) Union(others ...WordSet) WordSet {
	if len(s) == 0 && (len(others) == 0 || len(others[0]) == 0) {
		return nil
	}
	ret := make(WordSet)
	for w := range s {
		ret.Add(w)
	}
	for _, os := range others {
		for w := range os {
			ret.Add(w)
		}
	}
	return ret
}

// ProperNouns returns the set of words within the receiver that are proper nouns.
func (s WordSet) ProperNouns() WordSet {
	ret := make(WordSet)
	for w := range s {
		if w.IsProperNoun() {
			ret.Add(w)
		}
	}
	return ret
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
	if len(s) == 0 {
		panic("ChooseOneRandom on empty WordSet")
	}
	ofs := rand.Int() % len(s)
	i := 0
	for w := range s {
		if i == ofs {
			return w
		}
		i++
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
	// We parse all text in lowercase, because the POS tagger will use case
	// to identify proper nouns and so if we were to provide correctly-cased
	// text sometimes we would need to provide it every time to get consistent
	// results. Instead, we just accept that the tagger will therefore rarely
	// actually detect proper nouns in exchange for more consistency of tagging
	// with conversational sentences that tend to not be capitalized.
	text = strings.ToLower(text)

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
		sentences = append(sentences, fixupParsedSentence(sentence))
	}
	return sentences, nil
}

// fixupParsedSentence fixes some quirks of the tokenizer in the "prose"
// library where it produces non-ideal results. It applies its changes
// in-place, but returns the given sentence anyway for convenience.
func fixupParsedSentence(s Sentence) Sentence {
	// Despite claims in its documentation, the prose tokenizer doesn't
	// seem to properly handle open/close quotes, so we'll try to fix these
	// up here.
	const (
		double = '"'
		single = '\''
		open   = "``"
		close  = "''"
	)
	openQuotes := map[byte]bool{
		double: false,
		single: false,
	}

	for i, w := range s {
		// First we'll handle the given tags as documented, in case a
		// future version of prose starts handling them, so we don't
		// confuse ourselves here.
		switch w.Tag {
		case open:
			switch w.Text {
			case `"`, `“`:
				openQuotes[double] = true
			case `'`, `‘`:
				openQuotes[single] = true
			}
			continue
		case close:
			switch w.Text {
			case `"`, `”`:
				openQuotes[double] = false
			case `'`, `’`:
				openQuotes[single] = false
			}
			continue
		}

		// If we find a quote symbol without a quote tag then we'll fix
		// the tagging for it. Other tags around it will probably be wrong
		// too, sadly, but at least our stringification will get the whitespace
		// around quotes correct and we'll record the opens/closes properly
		// in our chains.
		switch w.Text {
		case `"`:
			if openQuotes[double] {
				s[i].Tag = close
			} else {
				s[i].Tag = open
			}
			openQuotes[double] = !openQuotes[double] // toggle
		case `'`: // this is safe because non-quote apostrophes get grouped in with other characters
			if openQuotes[single] {
				s[i].Tag = close
			} else {
				s[i].Tag = open
			}
			openQuotes[single] = !openQuotes[single] // toggle
		case `“`:
			s[i].Tag = open
			openQuotes[double] = true
		case `”`:
			s[i].Tag = close
			openQuotes[double] = false
		case `‘`:
			s[i].Tag = open
			openQuotes[single] = true
		case `’`:
			s[i].Tag = close
			openQuotes[single] = false
		}
	}

	return s
}
