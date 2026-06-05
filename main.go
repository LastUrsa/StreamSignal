package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	config := parseLaunchConfig(os.Args[1:])
	app := NewAppForMode(config.Mode)

	err := wails.Run(buildWailsOptions(app, config))
	if err != nil {
		println("Error:", err.Error())
	}
}

func buildWailsOptions(app *App, config launchConfig) *options.App {
	return &options.App{
		Title:       "StreamSignal",
		Width:       1280,
		Height:      860,
		MinWidth:    1100,
		MinHeight:   760,
		StartHidden: config.StartHidden,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 13, G: 21, B: 28, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: singleInstanceID(),
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				app.handleSecondInstanceLaunch(secondInstanceData.Args)
			},
		},
	}
}
