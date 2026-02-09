package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Request API 请求
type Request struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// Response API 响应
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// ActionHandler 动作处理函数类型
type ActionHandler func(action string, data json.RawMessage) (interface{}, error)

// Server HTTP API 服务器
type Server struct {
	handler     ActionHandler
	termHandler http.HandlerFunc
	vncHandler  http.HandlerFunc
	port        int
	apiKey      string
	version     string
}

// NewServer 创建 API 服务器
func NewServer(handler ActionHandler, termHandler, vncHandler http.HandlerFunc, port int, apiKey, version string) *Server {
	return &Server{
		handler:     handler,
		termHandler: termHandler,
		vncHandler:  vncHandler,
		port:        port,
		apiKey:      apiKey,
		version:     version,
	}
}

// Start 启动服务（阻塞）
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/api.json", s.handleAPI)
	mux.HandleFunc("/health", s.handleHealth)

	if s.termHandler != nil {
		mux.HandleFunc("/ws/terminal", s.termHandler)
	}
	if s.vncHandler != nil {
		mux.HandleFunc("/ws/vnc", s.vncHandler)
	}

	// 中间件链: CORS -> Auth -> Handler
	var handler http.Handler = mux
	if s.apiKey != "" {
		handler = APIKeyAuth(s.apiKey)(handler)
	}
	handler = CORSMiddleware(handler)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("VMCat API server listening on %s", addr)
	log.Printf("API Key: %s", s.apiKey)
	return http.ListenAndServe(addr, handler)
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Code: 405, Msg: "method not allowed"})
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Code: 400, Msg: "invalid request body"})
		return
	}

	data, err := s.handler(req.Action, req.Data)
	if err != nil {
		writeJSON(w, http.StatusOK, Response{Code: 1, Msg: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{Code: 0, Msg: "success", Data: data})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": s.version,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
