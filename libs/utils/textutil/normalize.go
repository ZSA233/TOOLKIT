package textutil

import (
	"bytes"
	"strings"
)

func NormalizeUTF8Lines(data []byte) string {
	text := string(bytes.ToValidUTF8(data, []byte("\uFFFD")))
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}
