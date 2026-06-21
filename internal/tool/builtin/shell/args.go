package shell

import (
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// IsBackgroundArgs reports bash tool args with run_in_background set.
func IsBackgroundArgs(rawArgs []byte) bool {
	args := tool.ArgsMap(rawArgs)
	bg, _ := args["run_in_background"].(bool)
	return bg
}
