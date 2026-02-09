package terminal

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	internalssh "vmcat/internal/ssh"

	"github.com/gorilla/websocket"
)

// Server WebSocket 终端服务
type Server struct {
	pool     *internalssh.Pool
	port     int
	listener net.Listener
}

// NewServer 创建终端服务
func NewServer(pool *internalssh.Pool) *Server {
	return &Server{pool: pool}
}

// Start 启动服务（随机端口监听 localhost）
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/ws/terminal", s.handleTerminal)
	mux.HandleFunc("/ws/vnc", s.handleVNC)

	go func() {
		if err := http.Serve(ln, mux); err != nil && !isClosedErr(err) {
			log.Printf("terminal server: %v", err)
		}
	}()

	log.Printf("terminal server on :%d", s.port)
	return nil
}

// Port 获取监听端口
func (s *Server) Port() int {
	return s.port
}

// Close 停止服务
func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// 桌面应用仅允许本机来源
		return origin == "" ||
			strings.HasPrefix(origin, "wails://") ||
			strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "http://127.0.0.1") ||
			strings.HasPrefix(origin, "https://localhost") ||
			strings.HasPrefix(origin, "https://127.0.0.1")
	},
}

// handleTerminal WebSocket 终端处理
func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	hostID := r.URL.Query().Get("host")
	if hostID == "" {
		http.Error(w, "missing host", 400)
		return
	}

	rows, _ := strconv.Atoi(r.URL.Query().Get("rows"))
	cols, _ := strconv.Atoi(r.URL.Query().Get("cols"))
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	client, err := s.pool.Get(hostID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 支持指定命令（如 virsh console）
	cmd := r.URL.Query().Get("cmd")
	shell, err := client.OpenShell(rows, cols, cmd)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		shell.Close()
		return
	}

	// 双向桥接 WebSocket <-> SSH Shell
	var once sync.Once
	done := make(chan struct{})

	cleanup := func() {
		once.Do(func() {
			close(done)
			conn.Close()
			shell.Close()
		})
	}

	// SSH stdout -> WebSocket
	go func() {
		defer cleanup()
		buf := make([]byte, 4096)
		for {
			n, err := shell.Stdout.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket -> SSH stdin
	go func() {
		defer cleanup()
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			switch msgType {
			case websocket.BinaryMessage, websocket.TextMessage:
				// resize 消息: 0x01 + rows(2 bytes) + cols(2 bytes)
				if len(data) == 5 && data[0] == 1 {
					newRows := int(data[1])<<8 | int(data[2])
					newCols := int(data[3])<<8 | int(data[4])
					shell.Resize(newRows, newCols)
					continue
				}
				if _, err := shell.Stdin.Write(data); err != nil {
					return
				}
			}
		}
	}()

	<-done
}

// handleVNC WebSocket -> SSH 隧道 -> VNC 代理
func (s *Server) handleVNC(w http.ResponseWriter, r *http.Request) {
	hostID := r.URL.Query().Get("host")
	portStr := r.URL.Query().Get("port")
	hostIP := r.URL.Query().Get("ip") // 宿主机 IP，用于避免 iptables 拦截 loopback

	if hostID == "" || portStr == "" {
		http.Error(w, "missing host or port", 400)
		return
	}

	vncPort, err := strconv.Atoi(portStr)
	if err != nil || vncPort <= 0 {
		http.Error(w, "invalid port", 400)
		return
	}

	client, err := s.pool.Get(hostID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// 通过 SSH 隧道连接远程 VNC 端口
	// 优先使用宿主机实际 IP（避免 iptables 拦截 127.0.0.1）
	addrs := []string{}
	if hostIP != "" {
		addrs = append(addrs, fmt.Sprintf("%s:%d", hostIP, vncPort))
	}
	addrs = append(addrs, fmt.Sprintf("127.0.0.1:%d", vncPort))

	var tcpConn net.Conn
	for _, addr := range addrs {
		tcpConn, err = client.Dial("tcp", addr)
		if err == nil {
			break
		}
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("dial vnc port %d: %v", vncPort, err), 500)
		return
	}

	// 读取 VNC 服务器握手验证连通性
	handshake := make([]byte, 12)
	tcpConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := tcpConn.Read(handshake)
	tcpConn.SetReadDeadline(time.Time{})
	if err != nil {
		tcpConn.Close()
		http.Error(w, fmt.Sprintf("vnc handshake failed: %v", err), 500)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		tcpConn.Close()
		return
	}

	// 把已读到的握手数据先发给 WebSocket 客户端
	if n > 0 {
		if werr := wsConn.WriteMessage(websocket.BinaryMessage, handshake[:n]); werr != nil {
			wsConn.Close()
			tcpConn.Close()
			return
		}
	}

	log.Printf("vnc proxy: host=%s port=%d", hostID, vncPort)

	// 双向桥接 WebSocket <-> TCP (VNC)
	var once sync.Once
	done := make(chan struct{})
	cleanup := func() {
		once.Do(func() {
			close(done)
			wsConn.Close()
			tcpConn.Close()
		})
	}

	// VNC -> WebSocket
	go func() {
		defer cleanup()
		buf := make([]byte, 32*1024)
		for {
			n, err := tcpConn.Read(buf)
			if n > 0 {
				if werr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// WebSocket -> VNC
	go func() {
		defer cleanup()
		for {
			_, data, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			if _, err := tcpConn.Write(data); err != nil {
				return
			}
		}
	}()

	<-done
}

func isClosedErr(err error) bool {
	return err != nil && (err == http.ErrServerClosed || strings.Contains(err.Error(), "use of closed network connection"))
}
