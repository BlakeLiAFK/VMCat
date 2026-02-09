package vm

import (
	"fmt"
	"path/filepath"
	"strings"
)

const defaultInstanceRoot = "/var/lib/libvirt/instances"

// InstanceDir 返回 instance 的目录路径
func InstanceDir(instanceRoot string, instanceID int) string {
	if instanceRoot == "" {
		instanceRoot = defaultInstanceRoot
	}
	return filepath.Join(instanceRoot, fmt.Sprintf("%d", instanceID))
}

// InitInstanceDir 在远程宿主机上创建 instance 目录结构
func (m *Manager) InitInstanceDir(hostID string, instanceRoot string, instanceID int) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	dir := InstanceDir(instanceRoot, instanceID)
	cmd := fmt.Sprintf("mkdir -p %s/iso", dir)
	if _, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("mkdir instance dir: %w", err)
	}
	return nil
}

// ValidateInstancePath 校验路径是否在 instance 目录内
func ValidateInstancePath(instanceRoot string, instanceID int, path string) error {
	dir := InstanceDir(instanceRoot, instanceID)
	clean := filepath.Clean(path)
	if !strings.HasPrefix(clean, dir+"/") && clean != dir {
		return fmt.Errorf("path %q is outside instance directory %q", path, dir)
	}
	return nil
}

// InstanceISOList 列出 instance 专属 ISO 目录中的文件
func (m *Manager) InstanceISOList(hostID string, instanceRoot string, instanceID int) ([]ISOFile, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	isoDir := filepath.Join(InstanceDir(instanceRoot, instanceID), "iso")
	cmd := fmt.Sprintf("find %s -maxdepth 1 -type f -name '*.iso' 2>/dev/null | sort", isoDir)
	output, err := client.Execute(cmd)
	if err != nil || strings.TrimSpace(output) == "" {
		return nil, nil
	}

	var files []ISOFile
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, ISOFile{
			Name: filepath.Base(line),
			Path: line,
		})
	}
	return files, nil
}
