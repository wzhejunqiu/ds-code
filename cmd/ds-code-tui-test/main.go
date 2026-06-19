//go:build tuitest

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/tuitest"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// 必须先隔离 HOME（testutil.NewIsolatedHome），再调用 logging.Setup 等会写 ~/.ds-code 的代码。
	stack, err := tuitest.NewHarness()
	if err != nil {
		return err
	}
	defer stack.Close()

	closeLog, err := logging.Setup(logging.Options{
		ProjectRoot: stack.Cfg.ProjectRoot,
		Verbosity:   0,
	})
	if err != nil {
		return err
	}
	defer closeLog()

	logging.L().Info("ds-code-tui-test: header notification auto-scroll demo enabled")

	input.TCaseRunner = tuitest.NewTCaseSubmit(stack.Registry)

	cmd := &cobra.Command{}
	return stack.App.RunTUIHarness(cmd, "", stack.Registry, harnessStartupNotices())
}
