package vm

import (
	"fmt"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// NetworkList 获取虚拟网络列表
func (m *Manager) NetworkList(hostID string) ([]Network, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}
	output, err := client.Execute("virsh net-list --all")
	if err != nil {
		return nil, fmt.Errorf("net-list: %w", err)
	}
	nets := parseNetList(output)
	// 补充 bridge 信息
	for i, n := range nets {
		infoOut, err := client.Execute(fmt.Sprintf("virsh net-info %s", internalssh.ShellQuote(n.Name)))
		if err == nil {
			info := parseDominfo(infoOut)
			nets[i].Bridge = info["Bridge"]
		}
	}
	return nets, nil
}

// NetworkStart 启动虚拟网络
func (m *Manager) NetworkStart(hostID, netName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh net-start %s", internalssh.ShellQuote(netName)))
	if err != nil {
		return fmt.Errorf("net-start: %s", output)
	}
	return nil
}

// NetworkStop 停止虚拟网络
func (m *Manager) NetworkStop(hostID, netName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh net-destroy %s", internalssh.ShellQuote(netName)))
	if err != nil {
		return fmt.Errorf("net-stop: %s", output)
	}
	return nil
}

// NetworkAutostart 设置虚拟网络自动启动
func (m *Manager) NetworkAutostart(hostID, netName string, enabled bool) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	flag := "--autostart"
	if !enabled {
		flag = "--no-autostart"
	}
	output, err := client.Execute(fmt.Sprintf("virsh net-autostart %s %s", internalssh.ShellQuote(netName), flag))
	if err != nil {
		return fmt.Errorf("net-autostart: %s", output)
	}
	return nil
}

// BridgeList 获取宿主机网桥列表
func (m *Manager) BridgeList(hostID string) ([]string, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}
	// 用 ip 命令获取网桥
	output, err := client.Execute("ip -o link show type bridge | awk -F': ' '{print $2}'")
	if err != nil {
		// 兜底: brctl show
		output, err = client.Execute("brctl show | tail -n +2 | awk '{print $1}'")
		if err != nil {
			return nil, fmt.Errorf("bridge-list: %w", err)
		}
	}
	var bridges []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			bridges = append(bridges, line)
		}
	}
	return bridges, nil
}

// parseNetList 解析 virsh net-list --all 输出
func parseNetList(output string) []Network {
	var nets []Network
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		n := Network{
			Name:  fields[0],
			State: fields[1],
		}
		if len(fields) >= 3 {
			n.Autostart = fields[2]
		}
		if len(fields) >= 4 {
			n.Persistent = fields[3]
		}
		nets = append(nets, n)
	}
	return nets
}
