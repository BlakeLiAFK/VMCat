package ssh

import (
	"fmt"
	"sync"
)

// Pool SSH 连接池，每台宿主机一个连接
type Pool struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewPool 创建连接池
func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*Client),
	}
}

// Get 获取指定宿主机的 SSH 客户端，不存在则返回错误
func (p *Pool) Get(hostID string) (*Client, error) {
	p.mu.RLock()
	client, ok := p.clients[hostID]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("host %s not connected", hostID)
	}
	return client, nil
}

// Connect 连接到宿主机
func (p *Pool) Connect(hostID string, cfg *Config) (*Client, error) {
	p.mu.Lock()
	// 已有连接则先关闭
	if old, ok := p.clients[hostID]; ok {
		old.Close()
		delete(p.clients, hostID)
	}
	p.mu.Unlock()

	client := NewClient(cfg)
	if err := client.Connect(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.clients[hostID] = client
	p.mu.Unlock()

	return client, nil
}

// Disconnect 断开宿主机连接
func (p *Pool) Disconnect(hostID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[hostID]; ok {
		client.Close()
		delete(p.clients, hostID)
	}
}

// IsConnected 检查宿主机是否已连接
func (p *Pool) IsConnected(hostID string) bool {
	p.mu.RLock()
	client, ok := p.clients[hostID]
	p.mu.RUnlock()

	if !ok {
		return false
	}
	return client.IsAlive()
}

// CloseAll 关闭所有连接
func (p *Pool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, client := range p.clients {
		client.Close()
		delete(p.clients, id)
	}
}
