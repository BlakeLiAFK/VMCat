package vm

import (
	"fmt"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// Manager VM 管理器
type Manager struct {
	pool *internalssh.Pool
}

// NewManager 创建 VM 管理器
func NewManager(pool *internalssh.Pool) *Manager {
	return &Manager{pool: pool}
}

// List 获取宿主机上所有虚拟机列表
func (m *Manager) List(hostID string) ([]VM, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	output, err := client.Execute("virsh list --all")
	if err != nil {
		return nil, fmt.Errorf("virsh list: %w", err)
	}

	vms := parseVMList(output)
	for i := range vms {
		vms[i].HostID = hostID
	}

	// 补充 CPU 和内存信息
	for i, vm := range vms {
		xmlOutput, err := client.Execute(fmt.Sprintf("virsh dumpxml %s", internalssh.ShellQuote(vm.Name)))
		if err != nil {
			continue
		}
		domain, err := parseDumpXML(xmlOutput)
		if err != nil {
			continue
		}
		vms[i].CPUs = domain.VCPU
		mem := domain.Memory.Value
		switch domain.Memory.Unit {
		case "KiB":
			vms[i].MemoryMB = mem / 1024
		case "GiB":
			vms[i].MemoryMB = mem * 1024
		case "bytes":
			vms[i].MemoryMB = mem / 1024 / 1024
		default:
			vms[i].MemoryMB = mem
		}
	}

	return vms, nil
}

// Get 获取虚拟机详情
func (m *Manager) Get(hostID, vmName string) (*VMDetail, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	// 获取 XML 配置
	xmlOutput, err := client.Execute(fmt.Sprintf("virsh dumpxml %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return nil, fmt.Errorf("virsh dumpxml: %w", err)
	}

	domain, err := parseDumpXML(xmlOutput)
	if err != nil {
		return nil, err
	}

	detail := domainToDetail(domain, hostID)

	// 获取运行状态
	infoOutput, err := client.Execute(fmt.Sprintf("virsh dominfo %s", internalssh.ShellQuote(vmName)))
	if err == nil {
		info := parseDominfo(infoOutput)
		detail.State = info["State"]
		if info["Autostart"] == "enable" {
			detail.Autostart = true
		}
	}

	// 获取 IP 地址
	ifOutput, err := client.Execute(fmt.Sprintf("virsh domifaddr %s", internalssh.ShellQuote(vmName)))
	if err == nil {
		ips := parseDomifaddr(ifOutput)
		for i, nic := range detail.NICs {
			mac := strings.ToLower(nic.MAC)
			if ip, ok := ips[mac]; ok {
				detail.NICs[i].IP = ip
			}
		}
	}

	return detail, nil
}

// Start 启动虚拟机
func (m *Manager) Start(hostID, vmName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh start %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh start: %s", output)
	}
	return nil
}

// Shutdown 优雅关闭虚拟机 (ACPI)
func (m *Manager) Shutdown(hostID, vmName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh shutdown %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh shutdown: %s", output)
	}
	return nil
}

// Destroy 强制关闭虚拟机
func (m *Manager) Destroy(hostID, vmName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh destroy %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh destroy: %s", output)
	}
	return nil
}

// Reboot 重启虚拟机
func (m *Manager) Reboot(hostID, vmName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh reboot %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh reboot: %s", output)
	}
	return nil
}

// Suspend 暂停虚拟机
func (m *Manager) Suspend(hostID, vmName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh suspend %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh suspend: %s", output)
	}
	return nil
}

// Resume 恢复虚拟机
func (m *Manager) Resume(hostID, vmName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh resume %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh resume: %s", output)
	}
	return nil
}

// Delete 删除虚拟机 (undefine)
func (m *Manager) Delete(hostID, vmName string, removeStorage bool) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("virsh undefine %s", internalssh.ShellQuote(vmName))
	if removeStorage {
		cmd += " --remove-all-storage"
	}
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("virsh undefine: %s", output)
	}
	return nil
}

// Rename 重命名虚拟机 (需关机状态)
func (m *Manager) Rename(hostID, oldName, newName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh domrename %s %s",
		internalssh.ShellQuote(oldName), internalssh.ShellQuote(newName)))
	if err != nil {
		return fmt.Errorf("virsh domrename: %s", output)
	}
	return nil
}

// SetVCPUs 设置 CPU 数量
func (m *Manager) SetVCPUs(hostID, vmName string, count int) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	q := internalssh.ShellQuote(vmName)
	// 先设置最大值 (--config)
	cmd := fmt.Sprintf("virsh setvcpus %s %d --config --maximum", q, count)
	if output, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("setvcpus max: %s", output)
	}
	// 再设置当前值 (--config)
	cmd = fmt.Sprintf("virsh setvcpus %s %d --config", q, count)
	if output, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("setvcpus config: %s", output)
	}
	return nil
}

// SetMemory 设置内存大小 (MB)
func (m *Manager) SetMemory(hostID, vmName string, sizeMB int) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	q := internalssh.ShellQuote(vmName)
	// 先设置最大内存 (--config)
	cmd := fmt.Sprintf("virsh setmaxmem %s %dM --config", q, sizeMB)
	if output, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("setmaxmem: %s", output)
	}
	// 再设置当前内存 (--config)
	cmd = fmt.Sprintf("virsh setmem %s %dM --config", q, sizeMB)
	if output, err := client.Execute(cmd); err != nil {
		return fmt.Errorf("setmem: %s", output)
	}
	return nil
}

// GetXML 获取 VM 的 XML 配置
func (m *Manager) GetXML(hostID, vmName string) (string, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return "", err
	}
	output, err := client.Execute(fmt.Sprintf("virsh dumpxml %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return "", fmt.Errorf("virsh dumpxml: %w", err)
	}
	return output, nil
}

// DefineXML 用 XML 定义/更新 VM
func (m *Manager) DefineXML(hostID, xmlContent string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	// 通过 stdin 传递 XML（ShellQuote 安全转义）
	cmd := fmt.Sprintf("echo %s | virsh define /dev/stdin", internalssh.ShellQuote(xmlContent))
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("virsh define: %s", output)
	}
	return nil
}

// Clone 克隆虚拟机
func (m *Manager) Clone(hostID, srcName, newName string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("virt-clone --original %s --name %s --auto-clone",
		internalssh.ShellQuote(srcName), internalssh.ShellQuote(newName))
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("virt-clone: %s", output)
	}
	return nil
}

// SetAutostart 设置自动启动
func (m *Manager) SetAutostart(hostID, vmName string, enabled bool) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	flag := "--autostart"
	if !enabled {
		flag = "--autostart --disable"
	}
	output, err := client.Execute(fmt.Sprintf("virsh autostart %s %s", flag, internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("virsh autostart: %s", output)
	}
	return nil
}
