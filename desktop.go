//go:build !headless

package main

import (
	"context"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"vmcat/internal/tray"
)

// runDesktop 桌面模式（Wails GUI）
func runDesktop() {
	app := NewApp()

	// 初始化系统托盘（主线程调用，必须在 wails.Run 之前）
	trayStart, trayEnd := tray.Init(&tray.Callbacks{
		OnShow: func() {
			if app.ctx == nil {
				return
			}
			wailsRuntime.WindowShow(app.ctx)
		},
		OnQuit: func() {
			app.forceQuit = true
			if app.ctx != nil {
				wailsRuntime.Quit(app.ctx)
			}
		},
	})
	trayStart()

	err := wails.Run(&options.App{
		Title:     "VMCat",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnBeforeClose: app.beforeClose,
		OnShutdown: func(ctx context.Context) {
			trayEnd()
			app.shutdown(ctx)
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
