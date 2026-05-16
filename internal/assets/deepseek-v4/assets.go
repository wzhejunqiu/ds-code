package deepseekv4

import _ "embed"

// TokenizerJSON is the HuggingFace tokenizer bundled for DeepSeek V4.
//
//go:embed tokenizer.json
var TokenizerJSON []byte
