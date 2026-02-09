package vm

import (
	"fmt"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// PoolList 获取存储池列表
func (m *Manager) PoolList(hostID string) ([]StoragePool, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}
	output, err := client.Execute("virsh pool-list --all --details")
	if err != nil {
		return nil, fmt.Errorf("pool-list: %w", err)
	}
	return parsePoolList(output), nil
}

// VolList 获取存储池中的卷列表
func (m *Manager) VolList(hostID, poolName string) ([]Volume, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}
	output, err := client.Execute(fmt.Sprintf("virsh vol-list %s --details", internalssh.ShellQuote(poolName)))
	if err != nil {
		return nil, fmt.Errorf("vol-list: %w", err)
	}
	return parseVolList(output), nil
}

// DeleteVolume 删除存储卷
func (m *Manager) DeleteVolume(hostID, poolName, volName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("virsh vol-delete --pool %s %s",
		internalssh.ShellQuote(poolName), internalssh.ShellQuote(volName))
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("vol-delete: %s", output)
	}
	return nil
}

// CreateVolume 在存储池中创建新卷
func (m *Manager) CreateVolume(hostID, poolName, volName string, sizeGB int, format string) (string, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return "", err
	}
	if format == "" {
		format = "qcow2"
	}
	cmd := fmt.Sprintf("virsh vol-create-as %s %s %dG --format %s",
		internalssh.ShellQuote(poolName), internalssh.ShellQuote(volName),
		sizeGB, internalssh.ShellQuote(format))
	output, err := client.Execute(cmd)
	if err != nil {
		return "", fmt.Errorf("vol-create-as: %s", output)
	}
	// 获取卷路径
	pathOutput, err := client.Execute(fmt.Sprintf("virsh vol-path %s --pool %s",
		internalssh.ShellQuote(volName), internalssh.ShellQuote(poolName)))
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(pathOutput), nil
}

// PoolStart 启动存储池
func (m *Manager) PoolStart(hostID, poolName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh pool-start %s", internalssh.ShellQuote(poolName)))
	if err != nil {
		return fmt.Errorf("pool-start: %s", output)
	}
	return nil
}

// PoolStop 停止存储池
func (m *Manager) PoolStop(hostID, poolName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh pool-destroy %s", internalssh.ShellQuote(poolName)))
	if err != nil {
		return fmt.Errorf("pool-stop: %s", output)
	}
	return nil
}

// PoolAutostart 设置存储池自动启动
func (m *Manager) PoolAutostart(hostID, poolName string, enabled bool) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	flag := "--autostart"
	if !enabled {
		flag = "--no-autostart"
	}
	output, err := client.Execute(fmt.Sprintf("virsh pool-autostart %s %s", internalssh.ShellQuote(poolName), flag))
	if err != nil {
		return fmt.Errorf("pool-autostart: %s", output)
	}
	return nil
}

// parsePoolList 解析 virsh pool-list --all --details 输出
func parsePoolList(output string) []StoragePool {
	var pools []StoragePool
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		p := StoragePool{
			Name:  fields[0],
			State: fields[1],
		}
		if len(fields) >= 3 {
			p.Autostart = fields[2]
		}
		if len(fields) >= 4 {
			p.Persistent = fields[3]
		}
		if len(fields) >= 5 {
			p.Capacity = fields[4]
		}
		if len(fields) >= 6 {
			p.Allocation = fields[5]
		}
		if len(fields) >= 7 {
			p.Available = fields[6]
		}
		pools = append(pools, p)
	}
	return pools
}

// parseVolList 解析 virsh vol-list --details 输出
// 格式: Name  Path  Type  Capacity  Allocation
func parseVolList(output string) []Volume {
	var vols []Volume
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
		v := Volume{
			Name: fields[0],
			Path: fields[1],
		}
		if len(fields) >= 3 {
			v.Type = fields[2]
		}
		// Capacity: "10.00 GiB"
		if len(fields) >= 5 {
			v.Capacity = fields[3] + " " + fields[4]
		} else if len(fields) >= 4 {
			v.Capacity = fields[3]
		}
		// Allocation: "196.00 KiB"
		if len(fields) >= 7 {
			v.Allocation = fields[5] + " " + fields[6]
		} else if len(fields) >= 6 {
			v.Allocation = fields[5]
		}
		vols = append(vols, v)
	}
	return vols
}
