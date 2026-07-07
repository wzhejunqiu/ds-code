//go:build !darwin

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func setupTray(_ *application.App, _ application.Window, _ *DesktopService) {}

func configureWindowClose(_ *application.App, _ application.Window) {}
