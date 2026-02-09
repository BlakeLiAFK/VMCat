package vm

import (
	"fmt"
	"strings"
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
	output, err := client.Execute(fmt.Sprintf("virsh vol-list %s --details", poolName))
	if err != nil {
		return nil, fmt.Errorf("vol-list: %w", err)
	}
	return parseVolList(output), nil
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
	cmd := fmt.Sprintf("virsh vol-create-as %s %s %dG --format %s", poolName, volName, sizeGB, format)
	output, err := client.Execute(cmd)
	if err != nil {
		return "", fmt.Errorf("vol-create-as: %s", output)
	}
	// 获取卷路径
	pathOutput, err := client.Execute(fmt.Sprintf("virsh vol-path %s --pool %s", volName, poolName))
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(pathOutput), nil
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
		if len(fields) >= 4 {
			v.Capacity = strings.Join(fields[3:], " ")
		}
		vols = append(vols, v)
	}
	return vols
}
