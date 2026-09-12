package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/cli"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 同一可执行文件双模式:带命令行参数时进入无头 CLI 模式
	// (项目探测/数据库/代码生成/AI/配置导出),否则启动图形界面。
	if len(os.Args) > 1 {
		os.Exit(cli.Run(os.Args[1:]))
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "GoWind Toolkit",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []any{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
