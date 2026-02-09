//go:build !headless

package main

import (
	"context"
	"log"

	"vmcat/internal/monitor"
	"vmcat/internal/store"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// wailsEmitter 桌面模式事件发射器
type wailsEmitter struct {
	ctx context.Context
}

func (e *wailsEmitter) Emit(ev string, data ...interface{}) {
	wailsRuntime.EventsEmit(e.ctx, ev, data...)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.emitter = &wailsEmitter{ctx: ctx}

	s, err := store.New()
	if err != nil {
		log.Printf("init store: %v", err)
		return
	}
	a.store = s

	// 迁移旧的明文密码为加密格式
	s.MigrateEncryptPasswords()

	// 启动终端 WebSocket 服务
	if err := a.termSrv.Start(); err != nil {
		log.Printf("start terminal server: %v", err)
	}

	// 启动资源历史采集器
	a.historyCollector = monitor.NewHistoryCollector(a.sshPool, a.store, a.monitor, a.vmManager)
	a.historyCollector.Start()
}

// beforeClose 拦截窗口关闭事件，隐藏到系统托盘
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.forceQuit {
		return false
	}
	wailsRuntime.WindowHide(ctx)
	return true
}
