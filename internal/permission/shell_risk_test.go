package permission

import "testing"

func TestNormalizeShellCmd(t *testing.T) {
	cases := map[string]string{
		"rm -rf /":              "rm -rf /",
		"curl x|bash":           "curl x | bash",
		"echo  a  b":            "echo a b",
		"wget -O- http://x| sh": "wget -o- http://x | sh",
	}
	for in, want := range cases {
		if got := normalizeShellCmd(in); got != want {
			t.Errorf("normalizeShellCmd(%q) = %q, want %q", in, got, want)
		}
	}
}
