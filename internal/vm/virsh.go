package vm

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// virsh 命令输出解析

// parseVMList 解析 virsh list --all 输出
func parseVMList(output string) []VM {
	var vms []VM
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Id") || strings.HasPrefix(line, "---") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		vm := VM{}

		// ID 可能是 - (未运行)
		if fields[0] != "-" {
			vm.ID, _ = strconv.Atoi(fields[0])
		}

		// 状态可能由多个词组成，如 "shut off"
		vm.Name = fields[1]
		vm.State = strings.Join(fields[2:], " ")
		vms = append(vms, vm)
	}
	return vms
}

// parseDominfo 解析 virsh dominfo 输出
func parseDominfo(output string) map[string]string {
	info := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			info[key] = val
		}
	}
	return info
}

// === XML 解析结构 ===

// DomainXML virsh dumpxml 的 XML 结构
type DomainXML struct {
	XMLName xml.Name        `xml:"domain"`
	Name    string          `xml:"name"`
	VCPU    int             `xml:"vcpu"`
	Memory  DomainMemory    `xml:"memory"`
	Devices DomainDevices   `xml:"devices"`
}

type DomainMemory struct {
	Value int    `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

type DomainDevices struct {
	Disks      []DomainDisk      `xml:"disk"`
	Interfaces []DomainInterface `xml:"interface"`
	Graphics   []DomainGraphics  `xml:"graphics"`
}

type DomainDisk struct {
	Device string           `xml:"device,attr"`
	Driver DomainDiskDriver `xml:"driver"`
	Source DomainDiskSource `xml:"source"`
	Target DomainDiskTarget `xml:"target"`
}

type DomainDiskDriver struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type DomainDiskSource struct {
	File string `xml:"file,attr"`
	Dev  string `xml:"dev,attr"`
}

type DomainDiskTarget struct {
	Dev string `xml:"dev,attr"`
}

type DomainInterface struct {
	Type   string               `xml:"type,attr"`
	MAC    DomainMAC            `xml:"mac"`
	Source DomainInterfaceSource `xml:"source"`
	Model  DomainModel          `xml:"model"`
}

type DomainMAC struct {
	Address string `xml:"address,attr"`
}

type DomainInterfaceSource struct {
	Bridge  string `xml:"bridge,attr"`
	Network string `xml:"network,attr"`
}

type DomainModel struct {
	Type string `xml:"type,attr"`
}

type DomainGraphics struct {
	Type   string `xml:"type,attr"`
	Port   int    `xml:"port,attr"`
	Listen string `xml:"listen,attr"`
}

// parseDumpXML 解析 virsh dumpxml 输出
func parseDumpXML(xmlData string) (*DomainXML, error) {
	var domain DomainXML
	if err := xml.Unmarshal([]byte(xmlData), &domain); err != nil {
		return nil, fmt.Errorf("parse xml: %w", err)
	}
	return &domain, nil
}

// domainToDetail 将 XML 数据转换为 VMDetail
func domainToDetail(domain *DomainXML, hostID string) *VMDetail {
	detail := &VMDetail{
		VM: VM{
			Name:   domain.Name,
			CPUs:   domain.VCPU,
			HostID: hostID,
		},
	}

	// 内存转换为 MB
	mem := domain.Memory.Value
	switch domain.Memory.Unit {
	case "KiB":
		detail.MemoryMB = mem / 1024
	case "GiB":
		detail.MemoryMB = mem * 1024
	case "bytes":
		detail.MemoryMB = mem / 1024 / 1024
	default: // MiB
		detail.MemoryMB = mem
	}

	// 磁盘
	for _, d := range domain.Devices.Disks {
		if d.Device != "disk" {
			continue
		}
		path := d.Source.File
		if path == "" {
			path = d.Source.Dev
		}
		detail.Disks = append(detail.Disks, Disk{
			Device: d.Target.Dev,
			Path:   path,
			Format: d.Driver.Type,
		})
	}

	// 网卡
	for _, iface := range domain.Devices.Interfaces {
		nic := NIC{
			MAC:     iface.MAC.Address,
			Bridge:  iface.Source.Bridge,
			Network: iface.Source.Network,
			Model:   iface.Model.Type,
		}
		detail.NICs = append(detail.NICs, nic)
	}

	// VNC 端口
	for _, g := range domain.Devices.Graphics {
		if g.Type == "vnc" && g.Port > 0 {
			detail.VNCPort = g.Port
			break
		}
	}

	return detail
}

// parseDomifaddr 解析 virsh domifaddr 输出，提取 IP
func parseDomifaddr(output string) map[string]string {
	// MAC -> IP 映射
	ips := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "---") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			mac := strings.ToLower(fields[1])
			ipWithMask := fields[3]
			ip := strings.Split(ipWithMask, "/")[0]
			ips[mac] = ip
		}
	}
	return ips
}
