//go:build darwin

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func applyExitConfirm(opts *application.Options, svc *DesktopService) {
	confirmer := newExitConfirmer(svc)
	opts.ShouldQuit = confirmer.shouldQuit
}
