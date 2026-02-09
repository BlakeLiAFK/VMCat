package vm

import "fmt"

// AttachDisk 添加磁盘
func (m *Manager) AttachDisk(hostID, vmName string, params DiskAttachParams) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	driver := params.Driver
	if driver == "" {
		driver = "qcow2"
	}
	cmd := fmt.Sprintf("virsh attach-disk %s %s %s --subdriver %s --persistent",
		vmName, params.Source, params.Target, driver)
	if params.Cache != "" {
		cmd += " --cache " + params.Cache
	}
	if params.DevType != "" {
		cmd += " --type " + params.DevType
	}
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("attach-disk: %s", output)
	}
	return nil
}

// DetachDisk 移除磁盘
func (m *Manager) DetachDisk(hostID, vmName, target string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh detach-disk %s %s --persistent", vmName, target))
	if err != nil {
		return fmt.Errorf("detach-disk: %s", output)
	}
	return nil
}

// AttachInterface 添加网卡
func (m *Manager) AttachInterface(hostID, vmName string, params NICAttachParams) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	nicType := params.Type
	if nicType == "" {
		nicType = "bridge"
	}
	model := params.Model
	if model == "" {
		model = "virtio"
	}
	cmd := fmt.Sprintf("virsh attach-interface %s %s %s --model %s --persistent",
		vmName, nicType, params.Source, model)
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("attach-interface: %s", output)
	}
	return nil
}

// DetachInterface 移除网卡
func (m *Manager) DetachInterface(hostID, vmName, macAddr string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	// 先尝试 bridge 类型
	cmd := fmt.Sprintf("virsh detach-interface %s bridge --mac %s --persistent", vmName, macAddr)
	output, err := client.Execute(cmd)
	if err != nil {
		// 再尝试 network 类型
		cmd = fmt.Sprintf("virsh detach-interface %s network --mac %s --persistent", vmName, macAddr)
		output, err = client.Execute(cmd)
		if err != nil {
			return fmt.Errorf("detach-interface: %s", output)
		}
	}
	return nil
}

// ChangeMedia 挂载光驱 ISO
func (m *Manager) ChangeMedia(hostID, vmName, target, source string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("virsh change-media %s %s %s --insert", vmName, target, source)
	output, err := client.Execute(cmd)
	if err != nil {
		return fmt.Errorf("change-media: %s", output)
	}
	return nil
}

// EjectMedia 弹出光驱
func (m *Manager) EjectMedia(hostID, vmName, target string) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("virsh change-media %s %s --eject", vmName, target))
	if err != nil {
		return fmt.Errorf("eject-media: %s", output)
	}
	return nil
}

// ResizeDisk 磁盘扩容 (qemu-img resize)
func (m *Manager) ResizeDisk(hostID, diskPath string, newSizeGB int) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("qemu-img resize %s %dG", diskPath, newSizeGB))
	if err != nil {
		return fmt.Errorf("qemu-img resize: %s", output)
	}
	return nil
}

// SetGraphics 设置 VNC 显示 (通过编辑 XML)
func (m *Manager) SetGraphics(hostID, vmName string, enabled bool) error {
	xmlStr, err := m.GetXML(hostID, vmName)
	if err != nil {
		return err
	}

	if enabled {
		// 检查是否已有 graphics
		domain, err := parseDumpXML(xmlStr)
		if err != nil {
			return err
		}
		for _, g := range domain.Devices.Graphics {
			if g.Type == "vnc" {
				return nil // 已存在
			}
		}
		// 在 </devices> 前插入 VNC graphics
		const vncXML = `    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'>
      <listen type='address' address='0.0.0.0'/>
    </graphics>
`
		insertPos := "</devices>"
		xmlStr = replaceFirst(xmlStr, insertPos, vncXML+insertPos)
	} else {
		// 移除所有 vnc graphics (简单方式: 解析再重写太复杂，用 sed 风格)
		client, err := m.pool.Get(hostID)
		if err != nil {
			return err
		}
		// 用 virt-xml 工具移除
		cmd := fmt.Sprintf("virt-xml %s --remove-device --graphics type=vnc", vmName)
		output, err := client.Execute(cmd)
		if err != nil {
			return fmt.Errorf("remove vnc: %s", output)
		}
		return nil
	}

	return m.DefineXML(hostID, xmlStr)
}

// replaceFirst 替换第一个匹配
func replaceFirst(s, old, new string) string {
	i := len(s)
	for j := 0; j <= len(s)-len(old); j++ {
		if s[j:j+len(old)] == old {
			i = j
			break
		}
	}
	if i == len(s) {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}
