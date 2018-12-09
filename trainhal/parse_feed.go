package trainhal

import (
	"fmt"
	"io"
	"strings"

	"github.com/apparentlymart/gopherhal/ghal"
	"github.com/mmcdole/gofeed"
)

func parseFeed(r io.Reader) ([]ghal.Sentence, error) {
	parser := gofeed.NewParser()
	feed, err := parser.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("error parsing feed: %s", err)
	}

	var ret []ghal.Sentence
	for _, item := range feed.Items {
		ss, _ := ghal.ParseText(item.Title)
		ret = append(ret, ss...)

		contentR := strings.NewReader(item.Content)
		ss, _ = parseHTMLFragment(contentR)
		ret = append(ret, ss...)

		contentR = strings.NewReader(item.Description)
		ss, _ = parseHTMLFragment(contentR)
		ret = append(ret, ss...)
	}
	return ret, nil
}
