//go:build darwin

package main

import (
	"fmt"
)

func pickFolderNative() (string, error) {
	if wailsApp == nil {
		return "", fmt.Errorf("application not ready")
	}
	path, err := wailsApp.Dialog.OpenFile().
		SetTitle("Open Project Folder").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}
