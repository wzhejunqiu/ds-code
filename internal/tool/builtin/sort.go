package builtin

import (
	"sort"
	"time"
)

// SortByModTimeDesc sorts items in place: ModTime descending, then path ascending.
func SortByModTimeDesc[T any](items []T, modTime func(T) time.Time, path func(T) string) {
	sort.Slice(items, func(i, j int) bool {
		ti, tj := modTime(items[i]), modTime(items[j])
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return path(items[i]) < path(items[j])
	})
}
