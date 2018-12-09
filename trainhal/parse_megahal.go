package trainhal

import (
	"bufio"
	"io"
	"strings"

	"github.com/apparentlymart/gopherhal/ghal"
)

func parseMegaHALTraining(r io.Reader) ([]ghal.Sentence, error) {
	sc := bufio.NewScanner(r)
	var ret []ghal.Sentence
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			// It's a comment, so ignore it.
			continue
		}
		sentences, _ := ghal.ParseText(line)
		ret = append(ret, sentences...)
	}
	return ret, nil
}
