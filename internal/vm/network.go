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

// NATRuleList 列出当前 iptables DNAT 规则
func (m *Manager) NATRuleList(hostID string) ([]NATRule, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}
	// 列出 nat 表 PREROUTING 链的 DNAT 规则
	output, err := client.Execute("sudo iptables -t nat -L PREROUTING -n --line-numbers 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("iptables list: %w", err)
	}
	var rules []NATRule
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "DNAT") {
			continue
		}
		rule := parseNATLine(line)
		if rule.VMIP != "" {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// NATRuleAdd 添加 DNAT 端口转发规则
func (m *Manager) NATRuleAdd(hostID, proto, hostPort, vmIP, vmPort, comment string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	if proto == "" {
		proto = "tcp"
	}
	// 添加 PREROUTING DNAT 规则
	cmd := fmt.Sprintf(
		"sudo iptables -t nat -A PREROUTING -p %s --dport %s -j DNAT --to-destination %s:%s",
		proto, hostPort, vmIP, vmPort,
	)
	if comment != "" {
		cmd += fmt.Sprintf(" -m comment --comment %s", internalssh.ShellQuote(comment))
	}
	if output, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("iptables add: %s", output)
	}
	// 添加 FORWARD 规则允许转发
	fwdCmd := fmt.Sprintf(
		"sudo iptables -C FORWARD -p %s -d %s --dport %s -j ACCEPT 2>/dev/null || sudo iptables -A FORWARD -p %s -d %s --dport %s -j ACCEPT",
		proto, vmIP, vmPort, proto, vmIP, vmPort,
	)
	client.Execute(fwdCmd)
	return nil
}

// NATRuleDelete 删除 DNAT 端口转发规则
func (m *Manager) NATRuleDelete(hostID, proto, hostPort, vmIP, vmPort string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	if proto == "" {
		proto = "tcp"
	}
	cmd := fmt.Sprintf(
		"sudo iptables -t nat -D PREROUTING -p %s --dport %s -j DNAT --to-destination %s:%s",
		proto, hostPort, vmIP, vmPort,
	)
	if output, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("iptables delete: %s", output)
	}
	// 尝试删除 FORWARD 规则
	fwdCmd := fmt.Sprintf(
		"sudo iptables -D FORWARD -p %s -d %s --dport %s -j ACCEPT 2>/dev/null",
		proto, vmIP, vmPort,
	)
	client.Execute(fwdCmd)
	return nil
}

// parseNATLine 解析 iptables DNAT 行
func parseNATLine(line string) NATRule {
	var rule NATRule
	fields := strings.Fields(line)
	for i, f := range fields {
		if f == "tcp" || f == "udp" {
			rule.Proto = f
		}
		// dpt:8080 or dpts:8080:8090
		if strings.HasPrefix(f, "dpt:") {
			rule.HostPort = strings.TrimPrefix(f, "dpt:")
		}
		if strings.HasPrefix(f, "dpts:") {
			rule.HostPort = strings.TrimPrefix(f, "dpts:")
		}
		// to:192.168.1.100:22
		if strings.HasPrefix(f, "to:") {
			dest := strings.TrimPrefix(f, "to:")
			// 格式: IP:port 或 IP:port-port
			if idx := strings.LastIndex(dest, ":"); idx > 0 {
				rule.VMIP = dest[:idx]
				rule.VMPort = dest[idx+1:]
			} else {
				rule.VMIP = dest
			}
		}
		// 注释
		if f == "/*" && i+1 < len(fields) {
			comment := ""
			for j := i + 1; j < len(fields); j++ {
				if fields[j] == "*/" {
					break
				}
				if comment != "" {
					comment += " "
				}
				comment += fields[j]
			}
			rule.Comment = comment
		}
	}
	return rule
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
