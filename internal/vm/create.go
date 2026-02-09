package vm

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Create 创建虚拟机 (virt-install)
func (m *Manager) Create(hostID string, params VMCreateParams) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	if params.Name == "" {
		return fmt.Errorf("VM name is required")
	}
	if params.CPUs <= 0 {
		params.CPUs = 1
	}
	if params.MemoryMB <= 0 {
		params.MemoryMB = 1024
	}

	cmd := fmt.Sprintf("virt-install --name %s --vcpus %d --memory %d",
		params.Name, params.CPUs, params.MemoryMB)

	// 磁盘
	if params.DiskPath != "" {
		disk := fmt.Sprintf("--disk path=%s", params.DiskPath)
		if params.DiskSizeGB > 0 {
			disk += fmt.Sprintf(",size=%d", params.DiskSizeGB)
		}
		cmd += " " + disk
	} else if params.DiskSizeGB > 0 {
		cmd += fmt.Sprintf(" --disk size=%d", params.DiskSizeGB)
	}

	// 光驱
	if params.CDROM != "" {
		cmd += fmt.Sprintf(" --cdrom %s", params.CDROM)
	} else {
		cmd += " --boot hd"
	}

	// 网络
	if params.Network != "" {
		netType := params.NetType
		if netType == "" {
			netType = "bridge"
		}
		if netType == "bridge" {
			cmd += fmt.Sprintf(" --network bridge=%s,model=virtio", params.Network)
		} else {
			cmd += fmt.Sprintf(" --network network=%s,model=virtio", params.Network)
		}
	} else {
		cmd += " --network default"
	}

	// 显示
	if params.VNC {
		cmd += " --graphics vnc,listen=0.0.0.0"
	} else {
		cmd += " --graphics none"
	}

	// OS 变体
	if params.OSVariant != "" {
		cmd += fmt.Sprintf(" --os-variant %s", params.OSVariant)
	}

	// 不进入交互式控制台
	cmd += " --noautoconsole"

	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("virt-install: %s", output)
	}
	return nil
}

// ISOList 列出宿主机上的 ISO 文件
func (m *Manager) ISOList(hostID string, searchPaths []string) ([]ISOFile, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	if len(searchPaths) == 0 {
		searchPaths = []string{"/var/lib/libvirt/images", "/home", "/root", "/tmp"}
	}

	pathStr := strings.Join(searchPaths, " ")
	cmd := fmt.Sprintf("find %s -maxdepth 3 -name '*.iso' -type f -printf '%%s %%p\\n' 2>/dev/null | head -100", pathStr)
	output, err := client.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("find iso: %w", err)
	}

	var isos []ISOFile
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		isos = append(isos, ISOFile{
			Name: filepath.Base(parts[1]),
			Path: parts[1],
			Size: formatBytes(parts[0]),
		})
	}
	return isos, nil
}

// OSVariantList 获取支持的 OS 变体列表
func (m *Manager) OSVariantList(hostID string) ([]string, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}
	output, err := client.Execute("osinfo-query os -f short-id | tail -n +3 | awk '{print $1}' | head -200")
	if err != nil {
		// osinfo-query 可能不存在，返回常用列表
		return defaultOSVariants(), nil
	}
	var variants []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			variants = append(variants, line)
		}
	}
	if len(variants) == 0 {
		return defaultOSVariants(), nil
	}
	return variants, nil
}

// formatBytes 格式化字节数
func formatBytes(sizeStr string) string {
	// sizeStr 是 find -printf %s 输出的字节数
	var size float64
	fmt.Sscanf(sizeStr, "%f", &size)
	switch {
	case size >= 1073741824:
		return fmt.Sprintf("%.1f GB", size/1073741824)
	case size >= 1048576:
		return fmt.Sprintf("%.1f MB", size/1048576)
	default:
		return fmt.Sprintf("%.0f KB", size/1024)
	}
}

// defaultOSVariants 常用 OS 变体列表
func defaultOSVariants() []string {
	return []string{
		"ubuntu22.04", "ubuntu20.04", "ubuntu18.04",
		"centos-stream9", "centos-stream8", "centos7.0",
		"debian12", "debian11", "debian10",
		"fedora39", "fedora38",
		"rocky9", "rocky8", "almalinux9", "almalinux8",
		"win10", "win11", "win2k22", "win2k19",
		"generic",
	}
}
