package lsp

import (
	"path/filepath"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/lsp/client"
)

// ServerConfig describes how to launch a language server.
type ServerConfig = client.ServerConfig

func defaultServers() map[string]ServerConfig {
	return map[string]ServerConfig{
		"go": {
			ID:         "go",
			Command:    "gopls",
			Args:       []string{"serve"},
			Extensions: []string{".go"},
		},
		"typescript": {
			ID:         "typescript",
			Command:    "typescript-language-server",
			Args:       []string{"--stdio"},
			Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"},
		},
		"cpp": {
			ID:         "cpp",
			Command:    "clangd",
			Args:       []string{},
			Extensions: []string{".c", ".h", ".cpp", ".hpp", ".cc", ".cxx", ".hxx"},
		},
		"java": {
			ID:         "java",
			Command:    "",
			Extensions: []string{".java"},
			Disabled:   true,
		},
		"rust": {
			ID:         "rust",
			Command:    "rust-analyzer",
			Args:       []string{},
			Extensions: []string{".rs"},
			Disabled:   true,
		},
		"python": {
			ID:         "python",
			Command:    "pyright-langserver",
			Args:       []string{"--stdio"},
			Extensions: []string{".py", ".pyi"},
			Disabled:   true,
		},
	}
}

// BuildRegistry merges built-in defaults with user LSP config.
func BuildRegistry(cfg config.LSPConfig) map[string]ServerConfig {
	out := defaultServers()
	for id, user := range cfg.Servers {
		base, ok := out[id]
		if !ok {
			base = ServerConfig{ID: id}
		}
		if user.Command != "" {
			base.Command = user.Command
		}
		if len(user.Args) > 0 {
			base.Args = append([]string(nil), user.Args...)
		}
		if len(user.Extensions) > 0 {
			base.Extensions = append([]string(nil), user.Extensions...)
		}
		if user.Env != nil {
			base.Env = user.Env
		}
		if user.Disabled {
			base.Disabled = true
		} else if user.Command != "" {
			base.Disabled = false
		}
		out[id] = base
	}
	return out
}

// ServerForExt returns the server ID for a file extension, or "".
func ServerForExt(registry map[string]ServerConfig, ext string) string {
	ext = strings.ToLower(ext)
	for id, srv := range registry {
		if srv.Disabled || srv.Command == "" {
			continue
		}
		for _, e := range srv.Extensions {
			if strings.ToLower(e) == ext {
				return id
			}
		}
	}
	return ""
}

// NormalizeExt ensures extension starts with '.'.
func NormalizeExt(path string) string {
	return strings.ToLower(filepath.Ext(path))
}
