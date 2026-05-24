package patch_test

import (
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/patch"
)

func TestFileLineStats_multiFile(t *testing.T) {
	text := `*** Begin Patch
*** Add File: new.go
+line1
+line2
*** Delete File: old.go
*** End Patch`
	stats, err := patch.FileLineStats(text, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("got %d stats", len(stats))
	}
	if stats[0].Added != 2 || stats[0].Removed != 0 {
		t.Fatalf("add file: %+v", stats[0])
	}
	if stats[1].Removed != 1 {
		t.Fatalf("delete: %+v", stats[1])
	}
}
