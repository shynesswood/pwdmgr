package main

import (
	"embed"
	"log"

	papp "pwdmgr/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := papp.NewApp()

	err := wails.Run(&options.App{
		Title:  "pwdmgr",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// 由前端按 prefers-color-scheme 调用 WindowSetBackgroundColour，避免与系统亮暗冲突
		Mac: &mac.Options{
			Appearance: mac.DefaultAppearance,
		},
		Windows: &windows.Options{
			Theme: windows.SystemDefault,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
