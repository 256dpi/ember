package ember

import (
	"regexp"
)

var unIndentPattern = regexp.MustCompile("\n\\s+")

func unIndent(str string) string {
	return unIndentPattern.ReplaceAllString(str, "\n")
}
