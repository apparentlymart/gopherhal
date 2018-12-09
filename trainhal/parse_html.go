package trainhal

import (
	"fmt"
	"io"
	"strings"

	"github.com/apparentlymart/gopherhal/ghal"
	"golang.org/x/net/html"
	htmla "golang.org/x/net/html/atom"
)

func parseHTML(r io.Reader) ([]ghal.Sentence, error) {
	node, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %s", err)
	}
	return extractHTMLNode(node), nil
}

func parseHTMLFragment(r io.Reader) ([]ghal.Sentence, error) {
	nodes, err := html.ParseFragment(r, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %s", err)
	}
	if anyHTMLNodesAreText(nodes) {
		// If we have direct text nodes at our root then that suggests
		// we're already inside a prose content element and so we'll
		// just slurp up all our text content.
		return extractHTMLNodesTextContent(nodes), nil
	}
	var ret []ghal.Sentence
	for _, node := range nodes {
		ret = append(ret, extractHTMLNode(node)...)
	}
	return ret, nil
}

func extractHTMLNode(node *html.Node) []ghal.Sentence {
	switch node.Type {
	case html.DocumentNode:
		return extractHTMLNodeChildren(node)
	case html.ElementNode:
		// What we'll do here depends on the element type:
		// - Some are considered effectively leaf elements that can't possibly
		//   contain any content, even recursively.
		// - Some are considered to be content containers, where any text
		//   nodes directly nested inside will have content extracted.
		// - For everything else we'll recursively visit child elements but
		//   ignore any direct-child text nodes.
		if isLeafHTMLElement(node) {
			return nil
		}
		switch node.DataAtom {
		case htmla.P, htmla.Li:
			// Direct child text nodes are probably content.
			return extractHTMLNodeTextContent(node)
		default:
			// For everything else, we'll just visit the child nodes.
			return extractHTMLNodeChildren(node)
		}
	}
	return nil
}

func extractHTMLNodeChildren(node *html.Node) []ghal.Sentence {
	var ret []ghal.Sentence
	node = node.FirstChild
	for node != nil {
		ret = append(ret, extractHTMLNode(node)...)
		node = node.NextSibling
	}
	return ret
}

func extractHTMLNodeTextContent(node *html.Node) []ghal.Sentence {
	var buf strings.Builder
	appendHTMLNodeTextContent(node, &buf)
	ss, _ := ghal.ParseText(buf.String())
	return ss
}

func extractHTMLNodesTextContent(nodes []*html.Node) []ghal.Sentence {
	var buf strings.Builder
	for _, node := range nodes {
		appendHTMLNodeTextContent(node, &buf)
	}
	ss, _ := ghal.ParseText(buf.String())
	return ss
}

func appendHTMLNodeTextContent(node *html.Node, buf *strings.Builder) {
	if isLeafHTMLElement(node) {
		return
	}
	switch node.Type {
	case html.TextNode:
		buf.WriteString(node.Data)
		buf.WriteByte(' ')
	case html.ElementNode:
		c := node.FirstChild
		for c != nil {
			appendHTMLNodeTextContent(c, buf)
			c = c.NextSibling
		}
	}
}

func isLeafHTMLElement(node *html.Node) bool {
	if node.Type != html.ElementNode {
		return false
	}
	switch node.DataAtom {
	case htmla.Script, htmla.Style, htmla.Frameset, htmla.Frame, htmla.Applet, htmla.Object, htmla.Form, htmla.Label, htmla.Pre, htmla.Plaintext, htmla.Listing, htmla.Menu, htmla.Table, htmla.Td, htmla.Tr, htmla.Th, htmla.Map, htmla.Noframes, htmla.Iframe, htmla.Picture, htmla.Img, htmla.Canvas, htmla.Svg, htmla.Video, htmla.Audio, htmla.Blockquote, htmla.Nav, htmla.Figure:
		// Skip leaf elements entirely; these are unlikely to contain prose content
		return true
	default:
		return false
	}
}

func anyHTMLNodesAreText(nodes []*html.Node) bool {
	for _, node := range nodes {
		if node.Type == html.TextNode {
			return true
		}
	}
	return false
}
