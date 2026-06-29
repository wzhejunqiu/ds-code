package version

import "testing"

func TestUserDataDirName(t *testing.T) {
	if UserDataDirName != ".ds-code" {
		t.Fatalf("UserDataDirName = %q, want .ds-code", UserDataDirName)
	}
}

func TestSystemPrefix(t *testing.T) {
	if SystemPrefix != "[ds-code] " {
		t.Fatalf("SystemPrefix = %q", SystemPrefix)
	}
}
