//go:build cgo

package deepseek

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../third_party/tokenizers

// Link flags (-ltokenizers -ldl -lm -lstdc++) come from github.com/daulet/tokenizers;
// only add the search path for third_party/tokenizers here to avoid duplicate -l warnings.
*/
import "C"
