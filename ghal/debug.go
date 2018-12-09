package ghal

import (
	"io"
	"log"
)

var debugLogger *log.Logger

func debugf(format string, args ...interface{}) {
	if debugLogger == nil {
		return
	}
	debugLogger.Printf(format, args...)
}

// SetDebugLog enables debug logging for this package, writing information
// to the given writer about how sentence construction is proceeding, etc.
//
// The exact format of this debug information is not part of the package
// interface and is subject to change in future releases.
func SetDebugLog(w io.Writer, prefix string) {
	debugLogger = log.New(w, prefix, 0)
}
