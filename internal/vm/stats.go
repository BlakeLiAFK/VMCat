package vm

import (
	"fmt"
	"strconv"
	"strings"
)

// VMStats 获取 VM 实时资源统计
func (m *Manager) VMStats(hostID, vmName string) (*VMResourceStats, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	output, err := client.Execute(fmt.Sprintf("virsh domstats %s --raw", vmName))
	if err != nil {
		return nil, fmt.Errorf("domstats: %w", err)
	}

	return parseDomstats(output), nil
}

// parseDomstats 解析 virsh domstats --raw 输出
func parseDomstats(output string) *VMResourceStats {
	stats := &VMResourceStats{}
	kv := make(map[string]string)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Domain:") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			kv[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	stats.CPUTime = parseUint64(kv["cpu.time"])
	stats.MemActual = parseUint64(kv["balloon.current"]) * 1024 // KiB -> bytes
	stats.MemRSS = parseUint64(kv["balloon.rss"]) * 1024

	// 聚合所有网卡
	for i := 0; ; i++ {
		rxKey := fmt.Sprintf("net.%d.rx.bytes", i)
		txKey := fmt.Sprintf("net.%d.tx.bytes", i)
		if _, ok := kv[rxKey]; !ok {
			break
		}
		stats.NetRxBytes += parseUint64(kv[rxKey])
		stats.NetTxBytes += parseUint64(kv[txKey])
	}

	// 聚合所有块设备
	for i := 0; ; i++ {
		rdKey := fmt.Sprintf("block.%d.rd.bytes", i)
		wrKey := fmt.Sprintf("block.%d.wr.bytes", i)
		if _, ok := kv[rdKey]; !ok {
			break
		}
		stats.BlockRdBytes += parseUint64(kv[rdKey])
		stats.BlockWrBytes += parseUint64(kv[wrKey])
	}

	return stats
}

func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
