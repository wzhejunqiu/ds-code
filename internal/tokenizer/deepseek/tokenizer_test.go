package deepseek

import (
	"testing"
)

func TestEncodeHello(t *testing.T) {
	tk, err := NewDefault()
	if err != nil {
		t.Fatalf("NewDefault: %v", err)
	}
	defer tk.Close()

	cases := []struct {
		text string
		want []uint32
	}{
		{"Hello!", []uint32{19923, 3}},
		{"Hello world", []uint32{19923, 2058}},
	}
	for _, tc := range cases {
		got := tk.Encode(tc.text)
		if len(got) != len(tc.want) {
			t.Fatalf("%q: got %v, want %v", tc.text, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%q: got %v, want %v", tc.text, got, tc.want)
			}
		}
	}
}
