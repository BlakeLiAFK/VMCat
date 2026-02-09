package monitor

import (
	"fmt"
	"strconv"
	"strings"

	internalssh "vmcat/internal/ssh"
)

// HostStats 宿主机资源统计
type HostStats struct {
	CPUPercent  float64 `json:"cpuPercent"`
	MemTotal    int64   `json:"memTotal"`    // MB
	MemUsed     int64   `json:"memUsed"`     // MB
	MemPercent  float64 `json:"memPercent"`
	DiskTotal   int64   `json:"diskTotal"`   // GB
	DiskUsed    int64   `json:"diskUsed"`    // GB
	DiskPercent float64 `json:"diskPercent"`
	Uptime      string  `json:"uptime"`
	LoadAvg     string  `json:"loadAvg"`
}

// Collector 资源采集器
type Collector struct {
	pool *internalssh.Pool
}

// NewCollector 创建资源采集器
func NewCollector(pool *internalssh.Pool) *Collector {
	return &Collector{pool: pool}
}

// Collect 采集宿主机资源信息
func (c *Collector) Collect(hostID string) (*HostStats, error) {
	client, err := c.pool.Get(hostID)
	if err != nil {
		return nil, err
	}

	// 合并多条命令减少 SSH 会话开销
	cmd := `echo "===MEM===" && free -m && echo "===DISK===" && df -BG --total 2>/dev/null | grep '^total' && echo "===CPU===" && top -bn1 | grep '%Cpu' | head -1 && echo "===UPTIME===" && uptime`
	output, err := client.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("collect stats: %w", err)
	}

	return parseStats(output), nil
}

// parseStats 解析采集输出
func parseStats(output string) *HostStats {
	stats := &HostStats{}
	sections := splitSections(output)

	// 解析内存
	if mem, ok := sections["MEM"]; ok {
		for _, line := range strings.Split(mem, "\n") {
			if strings.HasPrefix(line, "Mem:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					stats.MemTotal, _ = strconv.ParseInt(fields[1], 10, 64)
					stats.MemUsed, _ = strconv.ParseInt(fields[2], 10, 64)
					if stats.MemTotal > 0 {
						stats.MemPercent = float64(stats.MemUsed) / float64(stats.MemTotal) * 100
					}
				}
			}
		}
	}

	// 解析磁盘
	if disk, ok := sections["DISK"]; ok {
		fields := strings.Fields(strings.TrimSpace(disk))
		if len(fields) >= 3 {
			stats.DiskTotal = parseGB(fields[1])
			stats.DiskUsed = parseGB(fields[2])
			if stats.DiskTotal > 0 {
				stats.DiskPercent = float64(stats.DiskUsed) / float64(stats.DiskTotal) * 100
			}
		}
	}

	// 解析 CPU (从 top 输出提取 idle 百分比)
	if cpu, ok := sections["CPU"]; ok {
		if idx := strings.Index(cpu, "id"); idx > 0 {
			part := strings.TrimSpace(cpu[:idx])
			parts := strings.Split(part, ",")
			if len(parts) > 0 {
				last := strings.TrimSpace(parts[len(parts)-1])
				idle, _ := strconv.ParseFloat(last, 64)
				stats.CPUPercent = 100 - idle
			}
		}
	}

	// 解析 uptime
	if up, ok := sections["UPTIME"]; ok {
		line := strings.TrimSpace(up)
		stats.Uptime = line
		if idx := strings.Index(line, "load average:"); idx >= 0 {
			stats.LoadAvg = strings.TrimSpace(line[idx+len("load average:"):])
		}
	}

	return stats
}

// splitSections 按分隔符拆分输出
func splitSections(output string) map[string]string {
	sections := make(map[string]string)
	current := ""
	var buf strings.Builder

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "===") && strings.HasSuffix(line, "===") {
			if current != "" {
				sections[current] = buf.String()
				buf.Reset()
			}
			current = strings.Trim(line, "= ")
		} else if current != "" {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	if current != "" {
		sections[current] = buf.String()
	}
	return sections
}

// parseGB 解析 "123G" 格式的容量值
func parseGB(s string) int64 {
	s = strings.TrimSuffix(s, "G")
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
