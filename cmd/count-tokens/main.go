// Command count-tokens prints token IDs for a string using the embedded DeepSeek V4 tokenizer.
//
// Usage (after scripts/fetch-tokenizers-lib.sh):
//
//	CGO_ENABLED=1 go run ./cmd/count-tokens "Hello!"
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/tokenizer/deepseek"
)

func main() {
	text := "Hello!"
	if len(os.Args) > 1 {
		text = strings.Join(os.Args[1:], " ")
	}
	tk, err := deepseek.NewDefault()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer tk.Close()
	ids := tk.Encode(text)
	fmt.Println(ids)
	fmt.Println("count:", len(ids))
}
