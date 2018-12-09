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
	wordChains map[Word]chainSet

	// chains is a set containing all of the chains this brain knows.
	chains chainSet

	// wordsAfter and wordsBefore describe which words can succeed or
	// precede (respectively) each chain.
	wordsAfter  map[chain]WordSet
	wordsBefore map[chain]WordSet

	// startChains and endChains are the chains that can start or end sentences,
	// respectively.
	startChains chainSet
	endChains   chainSet
}

// NewBrain allocates and returns a new, empty brain, devoid of knowledge and
// ready to learn.
func NewBrain() *Brain {
	return &Brain{
		wordChains:  make(map[Word]chainSet),
		chains:      make(chainSet),
		wordsAfter:  make(map[chain]WordSet),
		wordsBefore: make(map[chain]WordSet),
		startChains: make(chainSet),
		endChains:   make(chainSet),
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
		chn := makeChain(s[i : i+chainLen])
		b.chains.Add(chn)

		for _, w := range chn {
			if _, ok := b.wordChains[w]; !ok {
				b.wordChains[w] = make(chainSet)
			}
			b.wordChains[w].Add(chn)
		}

		if i == 0 {
			b.startChains.Add(chn)
		} else {
			// The previous word can precede this chain.
			if _, ok := b.wordsBefore[chn]; !ok {
				b.wordsBefore[chn] = make(WordSet)
			}
			b.wordsBefore[chn].Add(s[i-1])
		}

		if i == (maxIdx - 1) {
			b.endChains.Add(chn)
		} else {
			// The following word can succeed this chain.
			if _, ok := b.wordsAfter[chn]; !ok {
				b.wordsAfter[chn] = make(WordSet)
			}
			b.wordsAfter[chn].Add(s[i+chainLen])
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
	return b.makeSentence(w, false)
}

// MakeReply takes one or more sentences and constructs a sentence in reply
// to them. This method constructs a number of candidate sentences using keywords
// from the given sentence and then assigns each a relevance score based on
// matching keywords from the given sentence. It returns one of the sentences
// with the highest relevance score.
//
// It is possible that there will be no reply at all if the brain doesn't
// know anything about the words in the given sentence. This is particularly
// likely for smaller brains. In that case, the return value is a nil Sentence.
func (b *Brain) MakeReply(ss ...Sentence) Sentence {
	var allWords, nouns, properNouns WordSet
	for _, s := range ss {
		allWords = allWords.Union(s.Words())
		nouns = nouns.Union(s.Nouns())
		properNouns = properNouns.Union(s.ProperNouns())
	}

	keywords := properNouns
	if len(keywords) < 2 {
		// If there's only one proper noun in the sentences (likely) then we'll
		// add the regular nouns into the mix too just so the responses aren't
		// always so predictable when proper nouns are present. The priority
		// we give to proper nouns during scoring will still serve to prioritize
		// responses containing these, but there will be some small chance of
		// selecting a sentence about something else if it has enough similar
		// regular nouns.
		keywords = nouns
	}
	if len(keywords) == 0 {
		keywords = allWords
	}
	if len(keywords) == 0 {
		// Should never happen unless the input sentence is itself empty,
		// which is degenerate input.
		return nil
	}

	// We'll try to produce a sentence for each of our keywords to start,
	// and then we'll score those sentences by how many other
	ss = make([]Sentence, 0, len(keywords))
	for w := range keywords {
		s := b.MakeSentenceWithKeyword(w)
		if len(s) > 0 {
			ss = append(ss, s)
		}
	}

	var bestSentence Sentence
	var bestScore int
	for _, s := range ss {
		score := 0
		for _, w := range s {
			// The points assigned here are pretty arbitrary and just
			// intended to give priority to words from the original sentence,
			// extra priority to proper nouns, and highest priority to
			// proper nouns from the original sentence.
			if w.IsProperNoun() {
				score++
			}
			if nouns.Has(w) { // nouns from the original sentence
				score += 2
			}
			if properNouns.Has(w) { // proper nouns from the original sentence
				score += 3 // properNouns is a subset of nouns, so these really get 1 + 2 + 3 = 6 points
			}
		}
		if score > bestScore {
			bestScore = score
			bestSentence = s
		}
	}

	return bestSentence
}

// MakeQuestion constructs a random question sentence using all of the
// question-sentence-terminals the brain has learned. This could be used to
// try to change the subject if normal reply behavior fails.
//
// This method can itself return a nil sentence if the brain hasn't yet seen
// any sentences that terminate with a question mark.
func (b *Brain) MakeQuestion() Sentence {
	return b.makeSentence(QuestionMark, true)
}

func (b *Brain) makeSentence(w Word, mustBeEnd bool) Sentence {
	b.mut.RLock()
	defer b.mut.RUnlock()

	chains := b.wordChains[w]
	if len(chains) == 0 {
		// If we don't know the given word, we can't make a sentence.
		return nil
	}

	// We'll start from one selected "middle chain" and then gradually
	// build sequences of words pseudorandomly both before and after that
	// chain until we've got a complete sentence (starting and ending with
	// chains from startChains and endChains as appropriate).
	var middleChain chain
	var before []Word // Built in reverse order first, and then reversed
	var after []Word
	if mustBeEnd {
		// This case is trickier since we need to scan over potentially
		// multiple chains containing our keyword until we find one that
		// is both an end chain _and_ has the keyword at the end. This special
		// case is used only to match terminal punctuation like question marks,
		// and so we expect that _most_ chains containing these will meet
		// our criteria, and we'll only be skipping odd situations like
		// embedded quotations containing question marks.
		for c := range chains {
			if c[chainLen-1] != w || !b.endChains.Has(c) {
				continue
			}
			middleChain = c
			break
		}
	} else {
		// Things are simpler if the keyword can be anywhere.
		middleChain = chains.ChooseOneRandom()
	}

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
