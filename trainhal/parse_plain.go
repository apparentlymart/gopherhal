package trainhal

import (
	"fmt"
	"io"

	"github.com/apparentlymart/gopherhal/ghal"
	"golang.org/x/text/encoding"
)

func parsePlain(r io.Reader, maybeEnc encoding.Encoding) ([]ghal.Sentence, error) {
	return nil, fmt.Errorf("plain text training not yet supported")
}
