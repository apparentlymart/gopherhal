package ghal

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/vmihailenco/msgpack"
)

// LoadBrain reads a serialized brain from the given reader, which must
// be in the format created by Brain.Save.
func LoadBrain(r io.Reader) (*Brain, error) {
	var fb fBrain
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(src) < 4 || !bytes.Equal(src[:4], fMagic) {
		return nil, fmt.Errorf("not a brain file")
	}
	err = msgpack.Unmarshal(src[4:], &fb)
	if err != nil {
		return nil, fmt.Errorf("invalid brain file: %s", err)
	}

	if fb.ChainLen != chainLen {
		return nil, fmt.Errorf("wrong chain length %d; need %d", fb.ChainLen, chainLen)
	}

	ret := NewBrain()

	wordByIdx := func(i fIndex) Word {
		if int(i) >= len(fb.Words) || i < 0 {
			return Word{} // invalid
		}
		return Word{
			Text: fb.Words[i].Text,
			Tag:  fb.Words[i].Tag,
		}
	}

	for i, fc := range fb.Chains {
		if got, want := len(fc.Words), chainLen; got != want {
			return nil, fmt.Errorf("chain %d has wrong length %d; need %d", i, got, want)
		}
		var c chain
		for i, wi := range fc.Words {
			c[i] = wordByIdx(wi)
		}
		ret.chains.Add(c)
		for _, w := range c {
			if _, exists := ret.wordChains[w]; !exists {
				ret.wordChains[w] = make(chainSet)
			}
			ret.wordChains[w].Add(c)
		}
		if _, exists := ret.wordsAfter[c]; !exists {
			ret.wordsAfter[c] = make(WordSet)
		}
		if _, exists := ret.wordsBefore[c]; !exists {
			ret.wordsBefore[c] = make(WordSet)
		}

		for _, wi := range fc.WordsAfter {
			ret.wordsAfter[c].Add(wordByIdx(wi))
		}
		for _, wi := range fc.WordsBefore {
			ret.wordsBefore[c].Add(wordByIdx(wi))
		}

		if fc.CanStart {
			ret.startChains.Add(c)
		}
		if fc.CanEnd {
			ret.endChains.Add(c)
		}
	}

	return ret, nil
}

// Save writes a snapshot of the receiving brain's contents into the given
// writer in a binary format that can be reloaded later with LoadBrain.
func (b *Brain) Save(w io.Writer) error {
	b.mut.RLock()
	defer b.mut.RUnlock()

	var fb fBrain
	fb.ChainLen = chainLen
	fb.Chains = make([]fChain, 0, len(b.chains))
	fb.Words = make([]fWord, 0, len(b.wordChains))

	wordIdxs := map[Word]fIndex{}

	wordIdx := func(w Word) fIndex {
		wIdx, exists := wordIdxs[w]
		if !exists {
			wIdx = fIndex(len(fb.Words))
			wordIdxs[w] = wIdx
			fb.Words = append(fb.Words, fWord{
				Tag:  w.Tag,
				Text: w.Text,
			})
		}
		return wIdx
	}

	for c := range b.chains {
		var fc fChain
		wds := make(fIndices, chainLen)
		for i, w := range c {
			wds[i] = wordIdx(w)
		}
		fc.Words = wds
		for w := range b.wordsAfter[c] {
			fc.WordsAfter = append(fc.WordsAfter, wordIdx(w))
		}
		for w := range b.wordsBefore[c] {
			fc.WordsBefore = append(fc.WordsBefore, wordIdx(w))
		}
		fc.CanStart = b.startChains.Has(c)
		fc.CanEnd = b.endChains.Has(c)
		fb.Chains = append(fb.Chains, fc)
	}

	src, err := msgpack.Marshal(&fb)
	if err != nil {
		return err
	}
	_, err = w.Write(fMagic)
	if err != nil {
		return err
	}
	_, err = w.Write(src)
	return err
}

// LoadBrainFile is like LoadBrain but it first opens the given filename
// and then reads data from it.
func LoadBrainFile(filename string) (*Brain, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return LoadBrain(f)
}

// SaveFile is like Save but it creates a file with the given filename
// and then writes the data to it.
func (b *Brain) SaveFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return b.Save(f)
}

var fMagic = []byte{'Q', 'W', 'O', 'K'}

type fBrain struct {
	ChainLen int64 `msgpack:"chainLen"`

	// indices into these lists are used in the other structures to keep the
	// file format relatively compact, storing each distinct word and chain
	// only once in the file.
	Chains []fChain `msgpack:"chains"`
	Words  []fWord  `msgpack:"words"`
}

type fChain struct {
	Words       fIndices `msgpack:"w"`
	WordsAfter  fIndices `msgpack:"a"`
	WordsBefore fIndices `msgpack:"b"`

	CanStart bool `msgpack:"s"`
	CanEnd   bool `msgpack:"e"`
}

type fWord struct {
	Tag  string `msgpack:"a"`
	Text string `msgpack:"e"`
}

var (
	_ msgpack.Marshaler   = (*fWord)(nil)
	_ msgpack.Unmarshaler = (*fWord)(nil)
)

type fIndex int64

type fIndices []fIndex

func (w fWord) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal([...]string{w.Text, w.Tag})
}

func (w *fWord) UnmarshalMsgpack(src []byte) error {
	var v [2]string
	err := msgpack.Unmarshal(src, &v)
	if err != nil {
		return err
	}
	w.Text = v[0]
	w.Tag = v[1]
	return nil
}
