package permission

import "testing"

func TestTokenizeShellCmd(t *testing.T) {
	got := tokenizeShellCmd(`cat .env && echo "hi there"`)
	want := []string{"cat", ".env", "&&", "echo", "hi there"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token[%d] = %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
}

func TestCheckShellDenylistPaths_embeddedLiteral(t *testing.T) {
	root := t.TempDir()
	e := NewEngine("auto", root, true)
	err := e.checkShellDenylistPaths(`python3 -c "open('.env').read()"`)
	if err == nil {
		t.Fatal("expected deny for embedded .env")
	}
}
