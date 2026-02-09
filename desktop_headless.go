//go:build headless

package main

import "fmt"

// runDesktop headless 模式下不支持桌面模式
func runDesktop() {
	fmt.Println("Desktop mode is not available in headless build.")
	fmt.Println("Usage: vmcat serve [options]")
}
