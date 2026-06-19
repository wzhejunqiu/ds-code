package permission

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// shellPathLiteralRE finds sensitive path-like fragments inside larger tokens (e.g. python -c).
var shellPathLiteralRE = regexp.MustCompile(`(?i)(?:^|[\s'"=(;|&])([.]?(?:env(?:rc|\.[a-z0-9._-]+)?|aws(?:/[^\s'";|&)]*)?|ssh(?:/[^\s'";|&)]*)?|docker(?:/[^\s'";|&)]*)?|kube(?:/[^\s'";|&)]*)?|gnupg(?:/[^\s'";|&)]*)?|npmrc|pypirc|netrc)|(?:^|/)(?:credentials|secrets)(?:/[^\s'";|&)]*)?|id_(?:rsa|ed25519|ecdsa|dsa)[^\s'";|&)]*)`)

func (e *Engine) checkShellDenylistPaths(cmd string) error {
	for _, tok := range tokenizeShellCmd(cmd) {
		if err := e.checkShellToken(tok); err != nil {
			return err
		}
	}
	for _, lit := range extractQuotedStrings(cmd) {
		if err := e.scanShellLiterals(lit); err != nil {
			return err
		}
	}
	return e.scanShellLiterals(cmd)
}

func (e *Engine) checkShellToken(tok string) error {
	tok = strings.Trim(tok, `"'`)
	if tok == "" || isShellMetaToken(tok) {
		return nil
	}
	tok = strings.TrimRight(tok, ";,)&|")
	if err := e.checkPathCandidate(tok); err != nil {
		return err
	}
	return e.scanShellLiterals(tok)
}

func (e *Engine) scanShellLiterals(s string) error {
	for _, m := range shellPathLiteralRE.FindAllStringSubmatch(s, -1) {
		if len(m) < 2 {
			continue
		}
		lit := strings.TrimLeft(m[1], "/")
		if lit == "" {
			continue
		}
		if err := e.checkPathCandidate(lit); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) checkPathCandidate(rel string) error {
	if !looksLikePathCandidate(rel) {
		return nil
	}
	base := filepath.Base(rel)
	if isSensitiveBasename(base) {
		return fmt.Errorf("%w: shell must not access sensitive path", ErrDenied)
	}
	abs, err := e.ResolveAccessPath(rel, PathRead)
	if err != nil {
		if filepath.IsAbs(rel) || isOutsideWorkspaceErr(err) {
			return fmt.Errorf("%w: shell path not allowed: %s", ErrDenied, rel)
		}
		return nil
	}
	_ = abs
	return nil
}

func looksLikePathCandidate(rel string) bool {
	if rel == "" || isShellMetaToken(rel) {
		return false
	}
	if strings.HasPrefix(rel, "-") && !strings.HasPrefix(rel, "-.") {
		return false
	}
	if isSensitiveBasename(filepath.Base(rel)) {
		return true
	}
	if strings.ContainsAny(rel, `/\`) || strings.HasPrefix(rel, ".") {
		return true
	}
	return false
}

func isShellMetaToken(tok string) bool {
	switch tok {
	case "|", "||", "&&", ";", ">", ">>", "<", "2>", "2>>", "&", "(", ")", "{", "}":
		return true
	}
	if strings.HasPrefix(tok, "2>") || strings.HasPrefix(tok, "1>") {
		return true
	}
	return false
}

func tokenizeShellCmd(cmd string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle, inDouble := false, false
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
				if cur.Len() > 0 {
					tokens = append(tokens, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '\\' && i+1 < len(cmd) {
				i++
				cur.WriteByte(cmd[i])
			} else if c == '"' {
				inDouble = false
				if cur.Len() > 0 {
					tokens = append(tokens, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			inSingle = true
		case c == '"':
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			inDouble = true
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func extractQuotedStrings(cmd string) []string {
	var out []string
	inSingle, inDouble := false, false
	var cur strings.Builder
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
				if cur.Len() > 0 {
					out = append(out, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '\\' && i+1 < len(cmd) {
				i++
				cur.WriteByte(cmd[i])
			} else if c == '"' {
				inDouble = false
				if cur.Len() > 0 {
					out = append(out, cur.String())
					cur.Reset()
				}
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		}
	}
	return out
}
