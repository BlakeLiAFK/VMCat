package vm

import (
	"fmt"
	"strings"
)

// TemplateCreateParams 模板创建 VM 的参数
type TemplateCreateParams struct {
	VMName       string `json:"vmName"`
	InstanceID   int    `json:"instanceId"`
	InstanceRoot string `json:"instanceRoot"`
	// Flavor
	CPUs     int `json:"cpus"`
	MemoryMB int `json:"memoryMB"`
	DiskGB   int `json:"diskGB"`
	// Image
	BasePath  string `json:"basePath"`
	OSVariant string `json:"osVariant"`
	// Network
	NetType string `json:"netType"` // network | bridge
	NetName string `json:"netName"`
}

// CreateFromTemplate 基于模板创建 VM
// 流程: mkdir -> qemu-img create -b -> virt-install --import
func (m *Manager) CreateFromTemplate(hostID string, params *TemplateCreateParams) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	instDir := InstanceDir(params.InstanceRoot, params.InstanceID)
	systemDisk := instDir + "/system.qcow2"

	// 1. 创建 instance 目录
	if _, err := client.Execute(fmt.Sprintf("mkdir -p %s/iso", instDir)); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// 2. 基于基础镜像创建 CoW 差分盘
	qemuCmd := fmt.Sprintf(
		"qemu-img create -b %s -F qcow2 -f qcow2 %s %dG",
		params.BasePath, systemDisk, params.DiskGB,
	)
	if output, err := client.Execute(qemuCmd); err != nil {
		return fmt.Errorf("qemu-img create: %s", output)
	}

	// 3. 构建 virt-install 命令
	var args []string
	args = append(args, "virt-install")
	args = append(args, "--name", params.VMName)
	args = append(args, "--vcpus", fmt.Sprintf("%d", params.CPUs))
	args = append(args, "--memory", fmt.Sprintf("%d", params.MemoryMB))
	args = append(args, fmt.Sprintf("--disk path=%s,format=qcow2", systemDisk))

	// 网络
	if params.NetName != "" {
		if params.NetType == "bridge" {
			args = append(args, fmt.Sprintf("--network bridge=%s,model=virtio", params.NetName))
		} else {
			args = append(args, fmt.Sprintf("--network network=%s,model=virtio", params.NetName))
		}
	} else {
		args = append(args, "--network network=default,model=virtio")
	}

	// OS 变体
	if params.OSVariant != "" {
		args = append(args, "--os-variant", params.OSVariant)
	}

	// 使用 --import 模式直接从磁盘启动
	args = append(args, "--import", "--noautoconsole", "--graphics", "vnc,listen=0.0.0.0")

	cmd := strings.Join(args, " ")
	if output, err := client.Execute(cmd); err != nil {
		// 创建失败时清理
		client.Execute(fmt.Sprintf("rm -rf %s", instDir))
		return fmt.Errorf("virt-install: %s", output)
	}

	// 4. 写入元信息
	metadata := fmt.Sprintf(`{"instanceId":%d,"vmName":"%s","flavorCpus":%d,"flavorMemMB":%d,"flavorDiskGB":%d,"basePath":"%s","osVariant":"%s"}`,
		params.InstanceID, params.VMName, params.CPUs, params.MemoryMB, params.DiskGB, params.BasePath, params.OSVariant)
	client.Execute(fmt.Sprintf("echo '%s' > %s/metadata.json", metadata, instDir))

	return nil
}
