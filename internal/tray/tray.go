package tray

import (
	_ "embed"

	"github.com/energye/systray"
)

//go:embed icon.png
var iconData []byte

// Callbacks 托盘事件回调
type Callbacks struct {
	OnShow func() // 显示窗口
	OnQuit func() // 退出应用
}

// Init 初始化系统托盘，返回 start/end 函数
// start 必须在主线程调用（wails.Run 之前）
// end 在 Wails OnShutdown 中调用
func Init(cb *Callbacks) (start, end func()) {
	return systray.RunWithExternalLoop(func() {
		systray.SetIcon(iconData)
		systray.SetTooltip("VMCat")

		mShow := systray.AddMenuItem("Show VMCat", "Show the main window")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Exit", "Quit VMCat")

		// 将菜单附加到状态栏图标（左键/右键点击都显示菜单）
		systray.CreateMenu()

		mShow.Click(func() {
			if cb.OnShow != nil {
				cb.OnShow()
			}
		})

		mQuit.Click(func() {
			if cb.OnQuit != nil {
				cb.OnQuit()
			}
		})
	}, nil)
}
