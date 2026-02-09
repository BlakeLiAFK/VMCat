package vm

import (
	"fmt"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// Migrate 在线迁移 VM 到目标宿主机
func (m *Manager) Migrate(srcHostID, vmName, dstHostID string) error {
	srcClient, err := m.pool.Get(srcHostID)
	if err != nil {
		return fmt.Errorf("源宿主机未连接: %w", err)
	}

	dstClient, err := m.pool.Get(dstHostID)
	if err != nil {
		return fmt.Errorf("目标宿主机未连接: %w", err)
	}

	// 获取目标宿主机地址
	dstHost, err := dstClient.Execute("hostname -f 2>/dev/null || hostname")
	if err != nil {
		return fmt.Errorf("获取目标主机名: %w", err)
	}
	dstHost = strings.TrimSpace(dstHost)

	// 预检查: 目标宿主机连通性
	checkCmd := fmt.Sprintf("virsh -c qemu+ssh://%s/system list 2>&1 | head -3", internalssh.ShellQuote(dstHost))
	output, err := srcClient.Execute(checkCmd)
	if err != nil {
		return fmt.Errorf("无法从源宿主机连接到目标: %s (输出: %s)", err, output)
	}

	// 执行在线迁移
	migrateCmd := fmt.Sprintf(
		"virsh migrate --live --persistent --undefinesource %s qemu+ssh://%s/system 2>&1",
		internalssh.ShellQuote(vmName),
		internalssh.ShellQuote(dstHost),
	)

	output, err = srcClient.Execute(migrateCmd)
	if err != nil {
		return fmt.Errorf("迁移失败: %w (输出: %s)", err, output)
	}

	return nil
}

// MigrateOffline 离线迁移 VM (通过客户端中继，适用于网络隔离场景)
// 流程: 关机 -> 导出XML -> 流式复制磁盘 -> 在目标定义VM -> 在源删除
func (m *Manager) MigrateOffline(srcHostID, vmName, dstHostID string, onProgress func(step, detail string)) error {
	srcClient, err := m.pool.Get(srcHostID)
	if err != nil {
		return fmt.Errorf("source host not connected: %w", err)
	}
	dstClient, err := m.pool.Get(dstHostID)
	if err != nil {
		return fmt.Errorf("target host not connected: %w", err)
	}

	progress := func(step, detail string) {
		if onProgress != nil {
			onProgress(step, detail)
		}
	}

	// 1. 检查 VM 是否关机
	progress("check", "checking VM state")
	infoOut, err := srcClient.Execute(fmt.Sprintf("virsh dominfo %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("get VM info: %w", err)
	}
	info := parseDominfo(infoOut)
	if info["State"] != "shut off" {
		return fmt.Errorf("VM must be shut off for offline migration (current: %s)", info["State"])
	}

	// 2. 导出 XML
	progress("xml", "exporting VM definition")
	xmlOut, err := srcClient.Execute(fmt.Sprintf("virsh dumpxml %s", internalssh.ShellQuote(vmName)))
	if err != nil {
		return fmt.Errorf("dump XML: %w", err)
	}

	// 3. 解析磁盘路径
	domain, err := parseDumpXML(xmlOut)
	if err != nil {
		return fmt.Errorf("parse XML: %w", err)
	}

	// 收集磁盘文件
	type diskInfo struct {
		srcPath string
		dstPath string
	}
	var disks []diskInfo
	dstBase := "/var/lib/libvirt/images"
	for _, d := range domain.Devices.Disks {
		if d.Source.File == "" || d.Device == "cdrom" {
			continue
		}
		// 目标路径: /var/lib/libvirt/images/<vmName>_<filename>
		fileName := d.Source.File
		if idx := strings.LastIndex(fileName, "/"); idx >= 0 {
			fileName = fileName[idx+1:]
		}
		dstPath := fmt.Sprintf("%s/%s_%s", dstBase, vmName, fileName)
		disks = append(disks, diskInfo{srcPath: d.Source.File, dstPath: dstPath})
	}

	// 4. 流式复制每个磁盘
	for i, disk := range disks {
		progress("copy", fmt.Sprintf("copying disk %d/%d: %s", i+1, len(disks), disk.srcPath))

		// 获取源文件大小
		sizeOut, err := srcClient.Execute(fmt.Sprintf("stat -c %%s %s", internalssh.ShellQuote(disk.srcPath)))
		if err != nil {
			return fmt.Errorf("get disk size %s: %w", disk.srcPath, err)
		}
		sizeOut = strings.TrimSpace(sizeOut)

		// 通过 cat 从源读取，通过 cat 写入目标
		// 使用管道: src SSH cat -> client memory -> dst SSH cat
		srcSession, err := srcClient.GetSSHClient().NewSession()
		if err != nil {
			return fmt.Errorf("src session: %w", err)
		}

		srcStdout, err := srcSession.StdoutPipe()
		if err != nil {
			srcSession.Close()
			return fmt.Errorf("src stdout pipe: %w", err)
		}

		if err := srcSession.Start(fmt.Sprintf("cat %s", internalssh.ShellQuote(disk.srcPath))); err != nil {
			srcSession.Close()
			return fmt.Errorf("src cat start: %w", err)
		}

		// 确保目标目录存在
		dstClient.Execute(fmt.Sprintf("mkdir -p %s", internalssh.ShellQuote(dstBase)))

		dstSession, err := dstClient.GetSSHClient().NewSession()
		if err != nil {
			srcSession.Close()
			return fmt.Errorf("dst session: %w", err)
		}

		dstStdin, err := dstSession.StdinPipe()
		if err != nil {
			srcSession.Close()
			dstSession.Close()
			return fmt.Errorf("dst stdin pipe: %w", err)
		}

		if err := dstSession.Start(fmt.Sprintf("cat > %s", internalssh.ShellQuote(disk.dstPath))); err != nil {
			srcSession.Close()
			dstSession.Close()
			return fmt.Errorf("dst cat start: %w", err)
		}

		// 流式复制
		buf := make([]byte, 256*1024) // 256KB buffer
		var copied int64
		for {
			n, readErr := srcStdout.Read(buf)
			if n > 0 {
				if _, writeErr := dstStdin.Write(buf[:n]); writeErr != nil {
					srcSession.Close()
					dstSession.Close()
					return fmt.Errorf("write to dst: %w", writeErr)
				}
				copied += int64(n)
				// 每 10MB 报告进度
				if copied%(10*1024*1024) < int64(n) {
					mb := copied / (1024 * 1024)
					progress("copy", fmt.Sprintf("disk %d/%d: %d MB copied", i+1, len(disks), mb))
				}
			}
			if readErr != nil {
				break
			}
		}

		dstStdin.Close()
		srcSession.Wait()
		srcSession.Close()
		dstSession.Wait()
		dstSession.Close()

		progress("copy", fmt.Sprintf("disk %d/%d done: %d MB", i+1, len(disks), copied/(1024*1024)))
	}

	// 5. 修改 XML 中的磁盘路径并在目标定义
	progress("define", "defining VM on target")
	modifiedXML := xmlOut
	for _, disk := range disks {
		modifiedXML = strings.ReplaceAll(modifiedXML, disk.srcPath, disk.dstPath)
	}

	// 写入临时 XML 并定义
	tmpXML := fmt.Sprintf("/tmp/vmcat_migrate_%s.xml", vmName)
	if _, err := dstClient.Execute(fmt.Sprintf("cat > %s << 'VMCAT_EOF'\n%s\nVMCAT_EOF", tmpXML, modifiedXML)); err != nil {
		return fmt.Errorf("write XML to target: %w", err)
	}
	output, err := dstClient.Execute(fmt.Sprintf("virsh define %s", tmpXML))
	if err != nil {
		return fmt.Errorf("define VM on target: %s", output)
	}
	dstClient.Execute(fmt.Sprintf("rm -f %s", tmpXML))

	// 6. 在源删除
	progress("cleanup", "removing VM from source")
	srcClient.Execute(fmt.Sprintf("virsh undefine %s", internalssh.ShellQuote(vmName)))

	progress("done", "migration completed")
	return nil
}
