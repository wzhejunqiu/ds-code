package apply

import (
	"fmt"
	"github.com/hejunqiu/ds-code/internal/patch"
	"strings"
)

func applyChunk(lines []string, chunk patch.Chunk) ([]string, error) {
	if len(chunk.Old) == 0 && len(chunk.New) > 0 {
		if chunk.Context != "" {
			ctx, err := findContext(lines, chunk.Context)
			if err != nil {
				return nil, err
			}
			if ctx < 0 {
				return nil, fmt.Errorf("context not found: %q", chunk.Context)
			}
			out := append(append([]string{}, lines[:ctx+1]...), chunk.New...)
			return append(out, lines[ctx+1:]...), nil
		}
		return append(lines, chunk.New...), nil
	}

	var idx int
	var err error
	if chunk.Context != "" {
		ctx, ctxErr := findContext(lines, chunk.Context)
		if ctxErr != nil {
			return nil, ctxErr
		}
		if ctx < 0 {
			return nil, fmt.Errorf("context not found: %q", chunk.Context)
		}
		searchFrom := ctx
		idx, err = findSubsliceUnique(lines[searchFrom:], chunk.Old)
		if err != nil {
			return nil, err
		}
		if idx >= 0 {
			idx += searchFrom
		}
	} else {
		idx, err = findSubsliceUnique(lines, chunk.Old)
		if err != nil {
			return nil, err
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("hunk not found (%d old lines)", len(chunk.Old))
	}
	if chunk.EOF && idx+len(chunk.Old) != len(lines) {
		return nil, fmt.Errorf("EOF hunk must end at end of file")
	}
	out := append(append([]string{}, lines[:idx]...), chunk.New...)
	out = append(out, lines[idx+len(chunk.Old):]...)
	return out, nil
}

func findContext(lines []string, ctx string) (int, error) {
	ctx = strings.TrimSpace(ctx)
	if ctx == "" {
		return -1, nil
	}
	var exact []int
	for i, l := range lines {
		if strings.TrimSpace(l) == ctx {
			exact = append(exact, i)
		}
	}
	switch len(exact) {
	case 1:
		return exact[0], nil
	case 0:
		// fall through to unique substring match
	default:
		return -1, fmt.Errorf("ambiguous context %q: %d matches", ctx, len(exact))
	}
	var substr []int
	for i, l := range lines {
		if strings.Contains(l, ctx) {
			substr = append(substr, i)
		}
	}
	switch len(substr) {
	case 0:
		return -1, nil
	case 1:
		return substr[0], nil
	default:
		return -1, fmt.Errorf("ambiguous context %q: %d matches", ctx, len(substr))
	}
}

func findSubslice(haystack, needle []string) int {
	if len(needle) == 0 {
		return len(haystack)
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func findSubsliceUnique(haystack, needle []string) (int, error) {
	idx := findSubslice(haystack, needle)
	if idx < 0 {
		return -1, nil
	}
	if findSubslice(haystack[idx+len(needle):], needle) >= 0 {
		return -1, fmt.Errorf("ambiguous hunk: multiple matches")
	}
	return idx, nil
}
