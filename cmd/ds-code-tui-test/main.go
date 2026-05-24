//go:build tuitest

package main

import (
	"fmt"
	"os"

	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/tuitest"
	"github.com/wzhejunqiu/ds-code/internal/ui/tui/model/input"
	"github.com/spf13/cobra"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	closeLog, err := logging.Setup(logging.Options{Verbosity: 0})
	if err != nil {
		return err
	}
	defer closeLog()

	stack, err := tuitest.NewHarness()
	if err != nil {
		return err
	}
	defer stack.Close()

	input.TCaseRunner = tuitest.NewTCaseSubmit(stack.Registry)

	cmd := &cobra.Command{}
	return stack.App.RunTUIHarness(cmd, "", stack.Registry)
}
