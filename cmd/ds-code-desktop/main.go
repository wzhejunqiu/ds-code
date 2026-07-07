package main

import (
	"log"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wzhejunqiu/ds-code/desktop/assets"
	desktopbridge "github.com/wzhejunqiu/ds-code/desktop/bridge"
)

func init() {
	application.RegisterEvent[desktopbridge.AgentEventEnvelope](desktopbridge.EventTopic)
}

func main() {
	var wailsApp *application.App
	svc := newDesktopService(func(env desktopbridge.AgentEventEnvelope) {
		if wailsApp != nil {
			wailsApp.Event.Emit(desktopbridge.EventTopic, env)
		}
	})

	opts := application.Options{
		Name:        "ds-code",
		Description: "ds-code desktop agent",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets.Dist),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	}
	modifyOptionsForIOS(&opts)
	wailsApp = application.New(opts)

	projectRoot := "."
	if len(os.Args) > 1 {
		projectRoot = os.Args[1]
	}

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
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

	// Best-effort auto-open when a project path is passed on the command line.
	if projectRoot != "." {
		go func() {
			if err := svc.OpenProject(projectRoot); err != nil {
				log.Printf("auto open project: %v", err)
			}
		}()
	}

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
