package trainhal

import (
	"fmt"
	"io"
	"mime"
	"path/filepath"

	"github.com/apparentlymart/gopherhal/ghal"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
)

type fileFormat string

const (
	formatUnknown  fileFormat = ""
	formatFeed     fileFormat = "feed"
	formatHTML     fileFormat = "html"
	formatMarkdown fileFormat = "md"
	formatPlain    fileFormat = "txt"
	formatMegaHAL  fileFormat = "mhtrn"
)

// selectFormat tries to determine a file format and suggested character
// encoding for the given filename and media type. Either may be set, and
// if both are set then the mediaType has preference. If neither are set,
// the result is always formatUnknown.
//
// A character encoding is returned only if it can be determined from the
// mediaType. nil is returned if no particular encoding is selected, meaning
// that the caller must either assume some default based on the detected format
// or sniff the source bytes to try to detect the encoding probabalistically.
// Even if an encoding is returned, a particular file format may have its own
// mechanism for specifying or detecting character encoding, in which case
// the caller should ignore the detected encoding.
func selectFormat(filename, mediaType string) (fileFormat, encoding.Encoding) {
	if mediaType != "" {
		format, enc := selectFormatFromMediaType(mediaType)
		if format != formatUnknown {
			return format, enc
		}
	}
	if filename != "" {
		format := selectFormatFromFilename(filename)
		return format, nil
	}
	return formatUnknown, nil
}

func selectFormatFromMediaType(mediaType string) (fileFormat, encoding.Encoding) {
	mimeType, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return formatUnknown, nil
	}
	var enc encoding.Encoding
	if charset := params["charset"]; charset != "" {
		enc, _ = ianaindex.MIME.Encoding(charset)
	}

	switch mimeType {
	case "text/html":
		return formatHTML, enc
	case "text/markdown", "text/x-markdown":
		return formatMarkdown, enc
	case "application/rss", "text/rss", "application/atom+xml", "application/atom", "text/atom", "application/xml", "text/xml":
		// Not all XML is a feed, but since we don't support any other HTML
		// formats we'll optimistically expect a feed and let the feed parser
		// detect if it isn't.
		return formatFeed, enc
	case "text/plain":
		return formatPlain, enc
	default:
		return formatUnknown, enc
	}
}

func selectFormatFromFilename(filename string) fileFormat {
	ext := filepath.Ext(filename)
	if ext == "" {
		return formatUnknown
	}

	switch ext {
	case ".html", ".htm":
		return formatHTML
	case ".md":
		return formatMarkdown
	case ".rss", ".atom", ".xml":
		// Not all XML is a feed, but since we don't support any other HTML
		// formats we'll optimistically expect a feed and let the feed parser
		// detect if it isn't.
		return formatFeed
	case ".txt":
		return formatPlain
	case ".trn":
		// Assume the MegaHAL training input file format, which is line-oriented
		// input with support for comments.
		return formatMegaHAL
	default:
		return formatUnknown
	}
}

func parseSource(r io.Reader, format fileFormat, maybeEnc encoding.Encoding) ([]ghal.Sentence, error) {
	switch format {
	case formatHTML:
		return parseHTML(r)
	case formatMarkdown:
		return parseMarkdown(r)
	case formatFeed:
		return parseFeed(r)
	case formatPlain:
		return parsePlain(r, maybeEnc)
	case formatMegaHAL:
		return parseMegaHALTraining(r)
	default:
		return nil, fmt.Errorf("unknown file format")
	}
}
