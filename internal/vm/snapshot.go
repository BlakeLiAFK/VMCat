package vm

import (
	"fmt"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// SnapshotList 获取快照列表
func (m *Manager) SnapshotList(hostID, vmName string) ([]Snapshot, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	output, err := client.Execute(fmt.Sprintf("virsh snapshot-list %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return nil, fmt.Errorf("snapshot-list: %w", err)
	}

	return parseSnapshotList(output), nil
}

// SnapshotCreate 创建快照
func (m *Manager) SnapshotCreate(hostID, vmName, snapName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("virsh snapshot-create-as %s --name %s",
		internalssh.ShellQuote(vmName), internalssh.ShellQuote(snapName))
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("snapshot-create: %s", output)
	}
	return nil
}

// SnapshotDelete 删除快照
func (m *Manager) SnapshotDelete(hostID, vmName, snapName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("virsh snapshot-delete %s %s",
		internalssh.ShellQuote(vmName), internalssh.ShellQuote(snapName))
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("snapshot-delete: %s", output)
	}
	return nil
}

// SnapshotRevert 恢复到指定快照
func (m *Manager) SnapshotRevert(hostID, vmName, snapName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("virsh snapshot-revert %s %s",
		internalssh.ShellQuote(vmName), internalssh.ShellQuote(snapName))
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("snapshot-revert: %s", output)
	}
	return nil
}

// parseSnapshotList 解析 virsh snapshot-list 输出
func parseSnapshotList(output string) []Snapshot {
	var snapshots []Snapshot
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		snap := Snapshot{
			Name:      fields[0],
			CreatedAt: fields[1] + " " + fields[2],
		}
		if len(fields) >= 4 {
			snap.State = fields[len(fields)-1]
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots
}
