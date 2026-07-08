//go:build darwin

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func prependPlatformMenu(menu *application.Menu, app *application.App) {
	appSub := menu.AddSubmenu("ds-code")
	appSub.AddRole(application.About)
	appSub.AddSeparator()
	appSub.AddRole(application.ServicesMenu)
	appSub.AddSeparator()
	appSub.AddRole(application.Hide)
	appSub.AddRole(application.HideOthers)
	appSub.AddRole(application.UnHide)
	appSub.AddSeparator()
	appSub.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(_ *application.Context) {
		app.Event.Emit("desktop:action", map[string]string{"action": "open_settings"})
	})
	appSub.AddRole(application.Quit)
}
