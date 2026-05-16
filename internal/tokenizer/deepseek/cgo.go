//go:build cgo

package deepseek

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../third_party/tokenizers -ltokenizers -ldl -lm -lstdc++
*/
import "C"
