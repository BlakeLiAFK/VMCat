package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// APIKeyAuth API Key 认证中间件
func APIKeyAuth(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 健康检查不需要认证
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// 从 Header 获取 token
			token := ""
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token = strings.TrimPrefix(auth, "Bearer ")
			}
			// WebSocket 从 query param 获取
			if token == "" {
				token = r.URL.Query().Get("token")
			}

			if token != key {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(Response{Code: 401, Msg: "unauthorized"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware 跨域中间件（服务端模式允许远程访问）
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
