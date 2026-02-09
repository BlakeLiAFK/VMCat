package vm

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	internalssh "vmcat/internal/ssh"
)

// cpuSample 存储上次 CPU 采样数据
type cpuSample struct {
	cpuTime    uint64
	ts         time.Time
	vcpus      int
	lastAccess time.Time
}

// cpuCache 全局 CPU 采样缓存
var (
	cpuCacheMu  sync.Mutex
	cpuCacheMap = make(map[string]*cpuSample) // key: hostID/vmName
)

// VMStats 获取 VM 实时资源统计
func (m *Manager) VMStats(hostID, vmName string) (*VMResourceStats, error) {
	client, err := m.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	output, err := client.Execute(fmt.Sprintf("virsh domstats %s --raw", internalssh.ShellQuote(vmName)))
	if err != nil {
		return nil, fmt.Errorf("domstats: %w", err)
	}

	stats := parseDomstats(output)

	// 计算 CPU 使用率 (delta)
	cacheKey := hostID + "/" + vmName
	now := time.Now()

	cpuCacheMu.Lock()
	prev := cpuCacheMap[cacheKey]
	cpuCacheMap[cacheKey] = &cpuSample{
		cpuTime:    stats.CPUTime,
		ts:         now,
		vcpus:      stats.VCPUs,
		lastAccess: now,
	}
	// 清理超过 5 分钟未访问的条目
	for k, v := range cpuCacheMap {
		if now.Sub(v.lastAccess) > 5*time.Minute {
			delete(cpuCacheMap, k)
		}
	}
	cpuCacheMu.Unlock()

	if prev != nil {
		elapsed := now.Sub(prev.ts).Nanoseconds()
		if elapsed > 0 && prev.cpuTime > 0 {
			deltaCPU := float64(stats.CPUTime - prev.cpuTime)
			vcpus := stats.VCPUs
			if vcpus <= 0 {
				vcpus = 1
			}
			// cpu.time 单位为纳秒，除以经过时间和 vCPU 数得百分比
			stats.CPUPercent = (deltaCPU / float64(elapsed)) / float64(vcpus) * 100
			if stats.CPUPercent < 0 {
				stats.CPUPercent = 0
			}
			if stats.CPUPercent > 100 {
				stats.CPUPercent = 100
			}
		}
	}

	return stats, nil
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
	stats.VCPUs = int(parseUint64(kv["vcpu.current"]))
	if stats.VCPUs <= 0 {
		stats.VCPUs = int(parseUint64(kv["vcpu.maximum"]))
	}
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
