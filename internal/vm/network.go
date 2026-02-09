package vm

import (
	"fmt"
	"strings"
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
		infoOut, err := client.Execute(fmt.Sprintf("virsh net-info %s", n.Name))
		if err == nil {
			info := parseDominfo(infoOut)
			nets[i].Bridge = info["Bridge"]
		}
	}
	return nets, nil
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
