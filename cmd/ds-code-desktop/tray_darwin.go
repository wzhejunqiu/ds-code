//go:build darwin

package main

import (
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func setupTray(app *application.App, win application.Window, svc *DesktopService) {
	tray := app.SystemTray.New()
	if b, err := os.ReadFile("build/darwin/icons.icns"); err == nil && len(b) > 0 {
		tray.SetIcon(b)
	}
	tray.SetLabel("ds-code")

	menu := app.NewMenu()
	menu.Add("Show Window").OnClick(func(_ *application.Context) {
		win.Show()
		win.Focus()
	})
	menu.Add("New Chat").OnClick(func(_ *application.Context) {
		wsID := svc.ActiveWorkspaceID()
		if wsID == "" {
			return
		}
		if _, err := svc.CreateChat(wsID); err == nil {
			win.Show()
			win.Focus()
			app.Event.Emit("desktop:action", map[string]string{"action": "chat_created"})
		}
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	tray.OnClick(func() {
		if win.IsVisible() {
			win.Hide()
		} else {
			win.Show()
			win.Focus()
		}
	})
}

func configureWindowClose(_ *application.App, win application.Window) {
	win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		win.Hide()
	})
}
