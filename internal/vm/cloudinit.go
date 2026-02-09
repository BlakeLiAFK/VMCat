package vm

import (
	"fmt"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// CloudInitConfig cloud-init 配置
type CloudInitConfig struct {
	Hostname  string `json:"hostname"`
	User      string `json:"user"`
	Password  string `json:"password"`
	SSHKey    string `json:"sshKey"`
	UserData  string `json:"userData"` // 自定义 user-data YAML
}

// GenerateCloudInitISO 在宿主机上生成 cloud-init seed ISO
func (m *Manager) GenerateCloudInitISO(hostID string, outputPath string, cfg CloudInitConfig) error {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return err
	}

	// 构建 meta-data
	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", cfg.Hostname, cfg.Hostname)

	// 构建 user-data
	var userData string
	if cfg.UserData != "" {
		userData = cfg.UserData
	} else {
		var parts []string
		parts = append(parts, "#cloud-config")
		if cfg.Hostname != "" {
			parts = append(parts, fmt.Sprintf("hostname: %s", cfg.Hostname))
		}
		if cfg.User != "" || cfg.Password != "" || cfg.SSHKey != "" {
			parts = append(parts, "users:")
			parts = append(parts, fmt.Sprintf("  - name: %s", defaultStr(cfg.User, "user")))
			parts = append(parts, "    sudo: ALL=(ALL) NOPASSWD:ALL")
			parts = append(parts, "    shell: /bin/bash")
			if cfg.Password != "" {
				parts = append(parts, "    lock_passwd: false")
				parts = append(parts, fmt.Sprintf("    plain_text_passwd: %s", cfg.Password))
			}
			if cfg.SSHKey != "" {
				parts = append(parts, "    ssh_authorized_keys:")
				parts = append(parts, fmt.Sprintf("      - %s", cfg.SSHKey))
			}
		}
		if cfg.Password != "" {
			parts = append(parts, "ssh_pwauth: true")
		}
		userData = strings.Join(parts, "\n")
	}

	// 写入临时文件
	tmpDir := "/tmp/cloudinit-" + cfg.Hostname
	cmds := []string{
		fmt.Sprintf("mkdir -p %s", internalssh.ShellQuote(tmpDir)),
		fmt.Sprintf("cat > %s/meta-data << 'CIEOF'\n%s\nCIEOF", tmpDir, metaData),
		fmt.Sprintf("cat > %s/user-data << 'CIEOF'\n%s\nCIEOF", tmpDir, userData),
	}

	for _, cmd := range cmds {
		if _, err := client.Execute(cmd); err != nil {
			return fmt.Errorf("write cloud-init files: %w", err)
		}
	}

	// 生成 ISO (优先 cloud-localds，降级 genisoimage)
	genCmd := fmt.Sprintf(
		`if command -v cloud-localds >/dev/null 2>&1; then cloud-localds %s %s/user-data %s/meta-data; elif command -v genisoimage >/dev/null 2>&1; then genisoimage -output %s -volid cidata -joliet -rock %s/user-data %s/meta-data; else echo "no_tool"; exit 1; fi`,
		internalssh.ShellQuote(outputPath),
		tmpDir, tmpDir,
		internalssh.ShellQuote(outputPath),
		tmpDir, tmpDir,
	)

	output, err := client.Execute(genCmd)
	if err != nil {
		return fmt.Errorf("generate cloud-init ISO: %w (output: %s)", err, output)
	}
	if strings.Contains(output, "no_tool") {
		return fmt.Errorf("宿主机缺少 cloud-localds 或 genisoimage 工具")
	}

	// 清理临时文件
	client.Execute(fmt.Sprintf("rm -rf %s", internalssh.ShellQuote(tmpDir)))

	return nil
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
