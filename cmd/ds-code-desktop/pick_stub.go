//go:build !darwin

package main

import "fmt"

func pickFolderNative() (string, error) {
	return "", fmt.Errorf("folder picker is only supported on macOS")
}
