package ember

import (
	"io"
	"net/http"
	"regexp"
)

var unIndentPattern = regexp.MustCompile("\n\\s+")

func unIndent(str string) string {
	return unIndentPattern.ReplaceAllString(str, "\n")
}

func fetch(url string) (string, string, string) {
	res, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	// ensure body is closed
	defer res.Body.Close()

	// read body
	data, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	return string(data), res.Header.Get("Content-Type"), res.Header.Get("Content-Length")
}
