package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"vmcat/internal/api"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 检查是否是 serve 子命令
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		runServer(os.Args[2:])
		return
	}

	// 桌面模式
	runDesktop()
}

// runServer 服务端模式（无头 HTTP API）
func runServer(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 9600, "API server port")
	apiKey := fs.String("api-key", "", "API key for authentication (auto-generated if empty)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: vmcat serve [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	// 支持环境变量
	if *apiKey == "" {
		if envKey := os.Getenv("VMCAT_API_KEY"); envKey != "" {
			*apiKey = envKey
		}
	}
	if envPort := os.Getenv("VMCAT_PORT"); envPort != "" {
		fmt.Sscanf(envPort, "%d", port)
	}

	// 自动生成 API Key
	if *apiKey == "" {
		b := make([]byte, 24)
		rand.Read(b)
		*apiKey = "vmcat_sk_" + hex.EncodeToString(b)
		log.Printf("Generated API Key: %s", *apiKey)
	}

	// 初始化 App（不启动 Wails）
	app := NewApp()
	if err := app.InitForServe(); err != nil {
		log.Fatalf("init failed: %v", err)
	}

	// 创建 API 服务器
	srv := api.NewServer(
		app.dispatch,
		app.termSrv.HandleTerminal,
		app.termSrv.HandleVNC,
		*port,
		*apiKey,
		app.AppVersion(),
	)

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		app.Shutdown()
		os.Exit(0)
	}()

	// 启动 HTTP 服务（阻塞）
	if err := srv.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
