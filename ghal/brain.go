package ghal

import (
	"sync"
)

// Brain is the main type in this package, containing all of the state for a
// single instance of the chatbot.
type Brain struct {
	mut sync.RWMutex

	// wordChains is a map from each of the words this brain knows to
	// the chains containing those words.
	wordChains map[Word]ChainSet

	// chains is a set containing all of the chains this brain knows.
	chains ChainSet

	// wordsAfter and wordsBefore describe which words can succeed or
	// precede (respectively) each chain.
	wordsAfter  map[Chain]WordSet
	wordsBefore map[Chain]WordSet

	// startChains and endChains are the chains that can start or end sentences,
	// respectively.
	startChains ChainSet
	endChains   ChainSet
}

// NewBrain allocates and returns a new, empty brain, devoid of knowledge and
// ready to learn.
func NewBrain() *Brain {
	return &Brain{
		wordChains:  make(map[Word]ChainSet),
		chains:      make(ChainSet),
		wordsAfter:  make(map[Chain]WordSet),
		wordsBefore: make(map[Chain]WordSet),
		startChains: make(ChainSet),
		endChains:   make(ChainSet),
	}
}

// AddSentence teaches the brain about the given sentence, allowing parts of
// it to be used in constructing replies.
func (b *Brain) AddSentence(s Sentence) {
	if len(s) < chainLen {
		// We need at least enough words to make one chain.
		return
	}

	b.mut.Lock()
	defer b.mut.Unlock()

	maxIdx := len(s) - (chainLen - 1)
	for i := 0; i < maxIdx; i++ {
		chain := MakeChain(s[i : i+chainLen])
		b.chains.Add(chain)

		for _, w := range chain {
			if _, ok := b.wordChains[w]; !ok {
				b.wordChains[w] = make(ChainSet)
			}
			b.wordChains[w].Add(chain)
		}

		if i == 0 {
			b.startChains.Add(chain)
		} else {
			// The previous word can precede this chain.
			if _, ok := b.wordsBefore[chain]; !ok {
				b.wordsBefore[chain] = make(WordSet)
			}
			b.wordsBefore[chain].Add(s[i-1])
		}

		if i == (maxIdx - 1) {
			b.endChains.Add(chain)
		} else {
			// The following word can succeed this chain.
			if _, ok := b.wordsAfter[chain]; !ok {
				b.wordsAfter[chain] = make(WordSet)
			}
			b.wordsAfter[chain].Add(s[i+chainLen])
		}
	}
}

// AddSentences teaches the brain about all of the given sentences. This is
// like AddSentence but perhaps more convenient when loading training data.
func (b *Brain) AddSentences(ss []Sentence) {
	for _, s := range ss {
		b.AddSentence(s)
	}
}

// MakeSentenceWithKeyword constructs a new sentence containing the given
// keyword.
//
// Will return nil if no sentence can be constructed for the given keyword.
func (b *Brain) MakeSentenceWithKeyword(w Word) Sentence {
	chains := b.wordChains[w]
	if len(chains) == 0 {
		// If we don't know the given word, we can't make a sentence.
		return nil
	}

	// We'll start from one selected "middle chain" and then gradually
	// build sequences of words pseudorandomly both before and after that
	// chain until we've got a complete sentence (starting and ending with
	// chains from startChains and endChains as appropriate).
	middleChain := chains.ChooseOneRandom()
	var before []Word // Built in reverse order first, and then reversed
	var after []Word

	// First we will work backwards to the beginning of the sentence.
	current := middleChain
	for !b.startChains.Has(current) {
		// Choose randomly one word that has preceeded this chain before,
		// thus adding one more word to the beginning of our sentence and
		// selecting a new chain for the next iteration.
		newWord := b.wordsBefore[current].ChooseOneRandom() // must exist if not in startChains
		before = append(before, newWord)
		current.PushBefore(newWord)
	}

	// Now we'll work forwards to the end of the sentence, in the same way.
	current = middleChain
	for !b.endChains.Has(current) {
		// Choose randomly one word that has preceeded this chain before,
		// thus adding one more word to the beginning of our sentence and
		// selecting a new chain for the next iteration.
		newWord := b.wordsAfter[current].ChooseOneRandom() // must exist if not in endChains
		after = append(after, newWord)
		current.PushAfter(newWord)
	}

	wordCount := len(before) + len(middleChain) + len(after)
	ret := make(Sentence, 0, wordCount)
	for i := len(before) - 1; i >= 0; i-- { // the "before" sequence is in reverse order
		ret = append(ret, before[i])
	}
	ret = append(ret, middleChain[:]...)
	ret = append(ret, after...)
	return ret
}