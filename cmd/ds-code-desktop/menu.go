package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func setupMenu(app *application.App, svc *DesktopService) {
	menu := app.NewMenu()

	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("Open Folder…").SetAccelerator("Cmd+O").OnClick(func(_ *application.Context) {
		path, err := svc.PickFolder()
		if err != nil || path == "" {
			return
		}
		if _, err := svc.AddWorkspace(path); err != nil {
			app.Logger.Error("add workspace", "err", err)
		}
	})
	fileMenu.Add("New Chat").SetAccelerator("Cmd+N").OnClick(func(_ *application.Context) {
		wsID := svc.ActiveWorkspaceID()
		if wsID == "" {
			return
		}
		if _, err := svc.CreateChat(wsID); err != nil {
			app.Logger.Error("create chat", "err", err)
		}
		app.Event.Emit("desktop:action", map[string]string{"action": "chat_created"})
	})
	fileMenu.AddSeparator()

	viewMenu := menu.AddSubmenu("View")
	viewMenu.Add("Toggle Sidebar").SetAccelerator("Cmd+\\").OnClick(func(_ *application.Context) {
		app.Event.Emit("desktop:action", map[string]string{"action": "toggle_sidebar"})
	})
	viewMenu.Add("Toggle Inspector").SetAccelerator("Cmd+Alt+\\").OnClick(func(_ *application.Context) {
		app.Event.Emit("desktop:action", map[string]string{"action": "toggle_inspector"})
	})
	viewMenu.Add("Command Palette").SetAccelerator("Cmd+K").OnClick(func(_ *application.Context) {
		app.Event.Emit("desktop:action", map[string]string{"action": "open_command_palette"})
	})

	settingsItem := menu.Add("Settings…").SetAccelerator("Cmd+,")
	settingsItem.OnClick(func(_ *application.Context) {
		app.Event.Emit("desktop:action", map[string]string{"action": "open_settings"})
	})

	workspaceMenu := menu.AddSubmenu("Workspace")
	workspaceMenu.Add("Add Folder…").OnClick(func(_ *application.Context) {
		path, err := svc.PickFolder()
		if err != nil || path == "" {
			return
		}
		if _, err := svc.AddWorkspace(path); err != nil {
			app.Logger.Error("add workspace", "err", err)
		}
		app.Event.Emit("desktop:action", map[string]string{"action": "workspace_added"})
	})

	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("Check for Updates…").OnClick(func(_ *application.Context) {
		checkForUpdates()
	})

	app.Menu.Set(menu)
}
