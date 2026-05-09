package main

import (
	"io/fs"
	"runtime"

	projectapp "mtu-tuner/internal/app"
	"mtu-tuner/internal/core"
	bindingnetwork "mtu-tuner/internal/views/transports/wailsv3/api/network"
	bindingsettings "mtu-tuner/internal/views/transports/wailsv3/api/settings"
	bindingsystem "mtu-tuner/internal/views/transports/wailsv3/api/system"
	bindingtasks "mtu-tuner/internal/views/transports/wailsv3/api/tasks"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	dispatcher := &wailsEventDispatcher{}
	service, err := projectapp.NewDefaultService(runtime.GOOS, nil)
	if err != nil {
		return err
	}
	projectapp.SetDefaultService(service)
	assetFS, err := frontendAssetFS()
	if err != nil {
		return err
	}

	app := application.New(defaultAppOptions(assetFS, []application.Service{
		application.NewService(bindingsystem.NewService(dispatcher)),
		application.NewService(bindingsettings.NewService(dispatcher)),
		application.NewService(bindingnetwork.NewService(dispatcher)),
		application.NewService(bindingtasks.NewService(dispatcher)),
		application.NewService(NewShellService()),
	}))
	dispatcher.app = app

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     core.AppName,
		Width:     1320,
		Height:    840,
		MinWidth:  980,
		MinHeight: 680,
	})

	return app.Run()
}

func defaultAppOptions(assetFS fs.FS, services []application.Service) application.Options {
	return application.Options{
		Name:        core.AppName,
		Description: core.AppDescription,
		Mac: application.MacOptions{
			// `make mtu-tuner-gui-run` uses `go run`; closing the last window should end the app process and return the terminal.
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assetFS),
		},
		Services: services,
	}
}

type wailsEventDispatcher struct {
	app *application.App
}

func (dispatcher *wailsEventDispatcher) Emit(name string, payload any) error {
	if dispatcher == nil || dispatcher.app == nil {
		return nil
	}
	dispatcher.app.Event.Emit(name, payload)
	return nil
}
