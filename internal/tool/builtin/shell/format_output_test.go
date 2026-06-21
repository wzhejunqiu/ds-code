package shell

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatShellOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stdout  string
		stderr  string
		runErr  error
		want    string
		wantErr bool
	}{
		{
			name:   "stdout only",
			stdout: "hello\n",
			want:   "stdout:\nhello\n",
		},
		{
			name:   "stderr only",
			stderr: "warn\n",
			want:   "stderr:\nwarn\n",
		},
		{
			name:   "both streams",
			stdout: "out\n",
			stderr: "err\n",
			want:   "stdout:\nout\nstderr:\nerr\n",
		},
		{
			name: "no output success",
			want: ResultNoOutput,
		},
		{
			name:   "exit error appended",
			stdout: "partial\n",
			runErr: errors.New("exit status 1"),
			want:   "stdout:\npartial\n" + ResultExitPrefix + "exit status 1",
		},
		{
			name:   "only exit error",
			runErr: errors.New("signal: killed"),
			want:   ResultExitPrefix + "signal: killed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := formatShellOutput(tt.stdout, tt.stderr, tt.runErr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("formatShellOutput() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("formatShellOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatShellOutput_preservesStreamOrder(t *testing.T) {
	out, err := formatShellOutput("a", "b", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "stdout:\na") {
		t.Fatalf("stdout should come first: %q", out)
	}
	if !strings.Contains(out, "stderr:\nb") {
		t.Fatalf("stderr should follow stdout: %q", out)
	}
}
