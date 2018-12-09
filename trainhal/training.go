package trainhal

import (
	"fmt"
	"io"

	"github.com/apparentlymart/gopherhal/ghal"
)

// ParseTrainingInput attempts to extract sentences from the given byte stream
// by interpreting it as one of a number of text formats:
//
//     - HTML
//     - RSS or Atom with HTML body text
//     - Markdown
//     - Plain text
//
// It uses the given optional filename and mimeType to guess which parser to
// use. If both are given, the mimeType has precedence.
// If neither filename nor mimeType are set then it will fail, returning an error.
func ParseTrainingInput(r io.Reader, filename, mediaType string) ([]ghal.Sentence, error) {
	format, mimeEnc := selectFormat(filename, mediaType)
	if format == formatUnknown {
		return nil, fmt.Errorf("failed to detect file format from filename or media type")
	}

	return parseSource(r, format, mimeEnc)
}
