package patch_test

import (
	"testing"

	"github.com/hejunqiu/ds-code/internal/patch"
)

func TestPaths(t *testing.T) {
	text := `*** Begin Patch
*** Add File: a.go
+x
*** Delete File: old.go
*** End Patch`
	paths, err := patch.Paths(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
}
