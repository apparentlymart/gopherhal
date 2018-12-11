package trainhal

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/apparentlymart/gopherhal/ghal"
)

func parseJSONUtter(r io.Reader) ([]ghal.Sentence, error) {
	// "JSON Utter" is a special JSON format that has already-parsed,
	// pre-tagged sentences. This is a fast way to import training data
	// that was parsed in a separate preprocessing step.
	dec := json.NewDecoder(r)

	var ret []ghal.Sentence

	tok, err := dec.Token()
	if err != nil {
		return ret, nil
	}
	if tok != json.Delim('[') {
		return ret, fmt.Errorf("JSON does not have array at root")
	}
	for dec.More() {
		var sentence ghal.Sentence
		err = dec.Decode(&sentence)
		if err != nil {
			return ret, err
		}
		ret = append(ret, sentence)
	}
	return ret, nil
}
