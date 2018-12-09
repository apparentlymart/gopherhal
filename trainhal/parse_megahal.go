package trainhal

import (
	"fmt"
	"io"

	"github.com/apparentlymart/gopherhal/ghal"
)

func parseMegaHALTraining(r io.Reader) ([]ghal.Sentence, error) {
	return nil, fmt.Errorf("MegaHAL-style training files not yet supported")
}
