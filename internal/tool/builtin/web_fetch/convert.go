package web_fetch

import (
	md "github.com/JohannesKaufmann/html-to-markdown/v2"
)

func htmlToMarkdown(html string) (string, error) {
	return md.ConvertString(html)
}
