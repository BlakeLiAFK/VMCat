//go:build headless

package main

import "context"

func (a *App) startup(ctx context.Context) {
	// headless 模式不需要 Wails startup
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}
