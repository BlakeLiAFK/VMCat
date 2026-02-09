package ssh

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

// Client SSH 客户端封装
type Client struct {
	config       *Config
	client       *ssh.Client
	connectedKey ssh.PublicKey // 连接后获取到的服务端公钥
	mu           sync.Mutex
	closed       bool
}

// Config SSH 连接配置
type Config struct {
	Host      string
	Port      int
	User      string
	AuthType  string // key | password
	KeyPath   string
	Password  string
	HostKey   string // base64 编码的已知服务端公钥
	ProxyAddr string // SOCKS5 代理地址，空则直连
}

// NewClient 创建 SSH 客户端
func NewClient(cfg *Config) *Client {
	return &Client{config: cfg}
}

// Connect 建立 SSH 连接
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		c.client.Close()
	}

	authMethods, err := c.buildAuth()
	if err != nil {
		return fmt.Errorf("build auth: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		HostKeyCallback: c.buildHostKeyCallback(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	if c.config.ProxyAddr != "" {
		// 通过 SOCKS5 代理建立 TCP 连接
		dialer, err := proxy.SOCKS5("tcp", c.config.ProxyAddr, nil, proxy.Direct)
		if err != nil {
			return fmt.Errorf("socks5 proxy: %w", err)
		}
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("proxy dial %s: %w", addr, err)
		}
		// 在代理连接上建立 SSH 握手
		ncc, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
		if err != nil {
			conn.Close()
			return fmt.Errorf("ssh via proxy: %w", err)
		}
		c.client = ssh.NewClient(ncc, chans, reqs)
	} else {
		client, err := ssh.Dial("tcp", addr, sshConfig)
		if err != nil {
			return fmt.Errorf("ssh dial %s: %w", addr, err)
		}
		c.client = client
	}
	c.closed = false
	return nil
}

// Execute 执行远程命令
func (c *Client) Execute(cmd string) (string, error) {
	c.mu.Lock()
	if c.client == nil || c.closed {
		c.mu.Unlock()
		if err := c.Connect(); err != nil {
			return "", err
		}
		c.mu.Lock()
	}
	client := c.client
	c.mu.Unlock()

	session, err := client.NewSession()
	if err != nil {
		// 连接可能已断开，尝试重连
		if reconnErr := c.Connect(); reconnErr != nil {
			return "", fmt.Errorf("reconnect: %w", reconnErr)
		}
		c.mu.Lock()
		client = c.client
		c.mu.Unlock()
		session, err = client.NewSession()
		if err != nil {
			return "", fmt.Errorf("new session: %w", err)
		}
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return strings.TrimSpace(string(output)), err
}

// GetSSHClient 获取底层 SSH 客户端（用于高级操作如管道流式传输）
func (c *Client) GetSSHClient() *ssh.Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client
}

// IsAlive 检查连接是否存活
func (c *Client) IsAlive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || c.closed {
		return false
	}

	// 发送 keepalive 检测
	_, _, err := c.client.SendRequest("keepalive@vmcat", true, nil)
	return err == nil
}

// Close 关闭连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closed = true
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// Dial 通过 SSH 隧道连接远程地址
func (c *Client) Dial(network, addr string) (net.Conn, error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("ssh not connected")
	}
	return client.Dial(network, addr)
}

// buildHostKeyCallback 构建 TOFU 主机密钥验证回调
func (c *Client) buildHostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		c.connectedKey = key

		// 首次连接（无已知密钥），接受并记录
		if c.config.HostKey == "" {
			return nil
		}

		// 解码已存储的公钥
		knownBytes, err := base64.StdEncoding.DecodeString(c.config.HostKey)
		if err != nil {
			// 存储的密钥格式异常，视为首次连接
			return nil
		}

		// 比对公钥
		if !bytes.Equal(key.Marshal(), knownBytes) {
			knownFP := fingerprintSHA256(knownBytes)
			remoteFP := FingerprintSHA256(key)
			return fmt.Errorf(
				"SSH 主机密钥不匹配，可能存在中间人攻击!\n"+
					"已知指纹: %s\n远端指纹: %s\n"+
					"如果确认服务器已重装或更换密钥，请在主机详情页忘记旧指纹后重试",
				knownFP, remoteFP,
			)
		}

		return nil
	}
}

// ConnectedHostKey 返回连接后获取到的服务端公钥（base64 编码）
func (c *Client) ConnectedHostKey() string {
	if c.connectedKey == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(c.connectedKey.Marshal())
}

// FingerprintSHA256 计算公钥的 SHA256 指纹
func FingerprintSHA256(key ssh.PublicKey) string {
	h := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(h[:])
}

// fingerprintSHA256 从原始字节计算 SHA256 指纹
func fingerprintSHA256(raw []byte) string {
	h := sha256.Sum256(raw)
	return "SHA256:" + base64.StdEncoding.EncodeToString(h[:])
}

// buildAuth 构建认证方法
func (c *Client) buildAuth() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	switch c.config.AuthType {
	case "password":
		methods = append(methods, ssh.Password(c.config.Password))
	default: // key
		keyPath := c.config.KeyPath
		if keyPath == "" {
			keyPath = findDefaultKey()
		}
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyPath, err)
		}

		var signer ssh.Signer
		if c.config.Password != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(c.config.Password))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	return methods, nil
}

// ShellSession SSH Shell 会话（PTY）
type ShellSession struct {
	session *ssh.Session
	Stdin   io.WriteCloser
	Stdout  io.Reader
}

// Close 关闭会话
func (s *ShellSession) Close() {
	if s.Stdin != nil {
		s.Stdin.Close()
	}
	if s.session != nil {
		s.session.Close()
	}
}

// Resize 调整终端尺寸
func (s *ShellSession) Resize(rows, cols int) error {
	if s.session == nil {
		return fmt.Errorf("session closed")
	}
	return s.session.WindowChange(rows, cols)
}

// OpenShell 打开带 PTY 的 Shell 会话，可选传入要执行的命令
func (c *Client) OpenShell(rows, cols int, cmd ...string) (*ShellSession, error) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		return nil, fmt.Errorf("not connected")
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		return nil, fmt.Errorf("request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		return nil, err
	}

	if len(cmd) > 0 && cmd[0] != "" {
		if err := session.Start(cmd[0]); err != nil {
			session.Close()
			return nil, fmt.Errorf("start cmd: %w", err)
		}
	} else {
		if err := session.Shell(); err != nil {
			session.Close()
			return nil, fmt.Errorf("start shell: %w", err)
		}
	}

	return &ShellSession{session: session, Stdin: stdin, Stdout: stdout}, nil
}

// findDefaultKey 按优先级查找默认 SSH 私钥
func findDefaultKey() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		home + "/.ssh/id_ed25519",
		home + "/.ssh/id_rsa",
		home + "/.ssh/id_ecdsa",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return home + "/.ssh/id_rsa"
}
