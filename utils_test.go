package ember

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

func clean(str string) string {
	// parse html
	node, err := html.Parse(strings.NewReader(str))
	if err != nil {
		panic(err)
	}

	// trim document
	trim(node)

	// render html
	var buf bytes.Buffer
	err = html.Render(&buf, node)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func trim(node *html.Node) {
	// remove whitespace
	if node.Type == html.TextNode {
		node.Data = strings.TrimSpace(node.Data) + "\n"
	}

	// trim children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		trim(child)
	}
}
