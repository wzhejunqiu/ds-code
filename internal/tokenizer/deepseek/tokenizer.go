// Package deepseek counts tokens for DeepSeek V4 using the bundled HuggingFace
// tokenizer (internal/assets/deepseek-v4/tokenizer.json).
//
// Setup (once per machine): scripts/fetch-tokenizers-lib.sh
// Requires CGO and third_party/tokenizers/libtokenizers.a.
package deepseek

import (
	"fmt"
	"sync"

	deepseekv4 "github.com/hejunqiu/ds-code/internal/assets/deepseek-v4"

	"github.com/daulet/tokenizers"
)

// Tokenizer wraps the DeepSeek V4 tokenizer.
// Encode matches transformers.AutoTokenizer.encode(text) with default options.
type Tokenizer struct {
	tk *tokenizers.Tokenizer
}

var (
	defaultOnce sync.Once
	defaultTok  *Tokenizer
	defaultErr  error
)

// New loads tokenizer.json from dir when dir is non-empty; otherwise uses embedded assets.
func New(dir string) (*Tokenizer, error) {
	path, useFile, err := resolveTokenizerFile(dir)
	if err != nil {
		return nil, err
	}

	var tk *tokenizers.Tokenizer
	switch {
	case useFile:
		tk, err = tokenizers.FromFile(path)
		if err != nil {
			return nil, fmt.Errorf("deepseek tokenizer: load %s: %w", path, err)
		}
	default:
		if len(deepseekv4.TokenizerJSON) == 0 {
			return nil, fmt.Errorf("deepseek tokenizer: embedded %s is empty", AssetsDir+"/"+tokenizerFile)
		}
		tk, err = tokenizers.FromBytes(deepseekv4.TokenizerJSON)
		if err != nil {
			return nil, fmt.Errorf("deepseek tokenizer: load embedded assets: %w", err)
		}
	}
	return &Tokenizer{tk: tk}, nil
}

// NewDefault uses the embedded tokenizer (no filesystem lookup).
func NewDefault() (*Tokenizer, error) {
	return New("")
}

// Default returns a process-wide singleton (lazy). Do not Close the result.
func Default() (*Tokenizer, error) {
	defaultOnce.Do(func() {
		defaultTok, defaultErr = NewDefault()
	})
	return defaultTok, defaultErr
}

// Close releases native tokenizer resources.
func (t *Tokenizer) Close() {
	if t != nil && t.tk != nil {
		t.tk.Close()
		t.tk = nil
	}
}

// Encode returns token IDs for text (same as Python tokenizer.encode(text)).
func (t *Tokenizer) Encode(text string) []uint32 {
	if t == nil || t.tk == nil {
		return nil
	}
	ids, _ := t.tk.Encode(text, false)
	return ids
}

// Count returns len(Encode(text)).
func (t *Tokenizer) Count(text string) int {
	return len(t.Encode(text))
}

// VocabSize returns vocabulary size.
func (t *Tokenizer) VocabSize() uint32 {
	if t == nil || t.tk == nil {
		return 0
	}
	return t.tk.VocabSize()
}
