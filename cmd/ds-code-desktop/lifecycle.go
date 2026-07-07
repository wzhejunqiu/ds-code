package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

var wailsApp *application.App

func setupLifecycle(app *application.App, svc *DesktopService) {
	wailsApp = app

	app.OnShutdown(func() {
		svc.Close()
	})

	// Restore active workspace on startup.
	if id := svc.mgr.ActiveID(); id != "" {
		if _, err := svc.mgr.Ensure(id); err != nil {
			app.Logger.Warn("restore workspace failed", "id", id, "err", err)
		}
	}
}

func restoreWorkspacesOnStart(svc *DesktopService) {
	for _, ws := range svc.mgr.List() {
		if !ws.Valid {
			continue
		}
		if ws.Active {
			if _, err := svc.mgr.Ensure(ws.ID); err != nil {
				wailsApp.Logger.Warn("ensure active workspace", "id", ws.ID, "err", err)
			}
		}
	}
}
