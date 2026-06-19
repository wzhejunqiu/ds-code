package patch

import (
	"path/filepath"
)

// FileLineStat holds per-file add/remove counts for TUI display.
type FileLineStat struct {
	Path    string
	Added   int
	Removed int
}

// FileLineStats parses a patch and returns per-file line change stats.
func FileLineStats(text string, validate PathValidator) ([]FileLineStat, error) {
	changes, err := Parse(text, validate)
	if err != nil {
		return nil, err
	}
	out := make([]FileLineStat, 0, len(changes))
	for _, c := range changes {
		st := FileLineStat{Path: c.Path}
		switch c.Kind {
		case ChangeAdd:
			st.Added = len(c.AddLines)
		case ChangeDelete:
			st.Removed = 1
		case ChangeUpdate:
			for _, ch := range c.Chunks {
				st.Added += ch.Added
				st.Removed += ch.Removed
			}
		}
		if c.MoveTo != "" {
			// Move is represented on the source path stat; destination is a separate change entry if present.
			_ = c.MoveTo
		}
		out = append(out, st)
	}
	return out, nil
}

// DisplayBasename returns the basename of a patch file path for TUI labels.
func DisplayBasename(path string) string {
	if path == "" {
		return path
	}
	return filepath.Base(path)
}
