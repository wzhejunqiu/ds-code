package main

import (
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wzhejunqiu/ds-code/desktop/assets"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
	desktopsys "github.com/wzhejunqiu/ds-code/desktop/sys"
)

func init() {
	application.RegisterEvent[desktopbridge.AgentEventEnvelope](desktopbridge.EventTopic)
}

func main() {
	desktopsys.EnsurePATH()
	desktopsys.Hooks.UpdateBadge = setDockBadge
	desktopsys.Hooks.Notify = desktopsys.Notify

	var app *application.App
	emitDesktopAction := func(action string) {
		if app != nil {
			app.Event.Emit("desktop:action", map[string]string{"action": action})
		}
	}
	svc, err := newDesktopService(func(env desktopbridge.AgentEventEnvelope) {
		if app != nil {
			app.Event.Emit(desktopbridge.EventTopic, env)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	opts := application.Options{
		Name:        "ds-code",
		Description: "ds-code desktop agent",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets.Dist),
		},
		KeyBindings: map[string]func(application.Window){
			"Cmd+,": func(_ application.Window) { emitDesktopAction("open_settings") },
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	}
	modifyOptionsForIOS(&opts)
	applyExitConfirm(&opts, svc)
	app = application.New(opts)
	desktopsys.Hooks.BackgroundAgentComplete = func(workspaceID, sessionID, agentID string) {
		app.Event.Emit("desktop:focus_subagent", map[string]string{
			"workspaceId": workspaceID,
			"sessionId":   sessionID,
			"subagentId":  agentID,
		})
	}
	setupLifecycle(app, svc)
	setupMenu(app, svc)

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "ds-code",
		Width:  1100,
		Height: 780,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(15, 17, 23),
		URL:              "/",
	})
	setupTray(app, win, svc)
	configureWindowClose(app, win)

	projectRoot := "."
	if len(os.Args) > 1 {
		projectRoot = os.Args[1]
	}
	if projectRoot != "." {
		go func() {
			if err := svc.OpenProject(projectRoot); err != nil {
				log.Printf("auto open project: %v", err)
			}
		}()
	} else {
		restoreWorkspacesOnStart(svc)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
