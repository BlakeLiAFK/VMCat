package monitor

import (
	"log"
	"sync"
	"time"

	internalssh "vmcat/internal/ssh"
	"vmcat/internal/store"
	"vmcat/internal/vm"
)

// HistoryCollector 资源历史采集器
type HistoryCollector struct {
	pool      *internalssh.Pool
	store     *store.Store
	monitor   *Collector
	vmManager *vm.Manager
	stopCh    chan struct{}
	once      sync.Once
}

// NewHistoryCollector 创建历史采集器
func NewHistoryCollector(pool *internalssh.Pool, s *store.Store, monitor *Collector, vmManager *vm.Manager) *HistoryCollector {
	return &HistoryCollector{
		pool:      pool,
		store:     s,
		monitor:   monitor,
		vmManager: vmManager,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动定时采集（每 30 秒）
func (h *HistoryCollector) Start() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// 启动后清理旧数据
		h.store.StatsCleanup(24)

		for {
			select {
			case <-ticker.C:
				h.collectAll()
				// 每次采集后清理超过 24 小时的数据
				h.store.StatsCleanup(24)
			case <-h.stopCh:
				return
			}
		}
	}()
}

// Stop 停止采集
func (h *HistoryCollector) Stop() {
	h.once.Do(func() {
		close(h.stopCh)
	})
}

// collectAll 采集所有已连接宿主机的资源数据
func (h *HistoryCollector) collectAll() {
	hosts, err := h.store.HostList()
	if err != nil {
		return
	}

	for _, host := range hosts {
		if !h.pool.IsConnected(host.ID) {
			continue
		}

		// 采集宿主机资源
		stats, err := h.monitor.Collect(host.ID)
		if err != nil {
			log.Printf("history collect host %s: %v", host.ID, err)
			continue
		}

		if err := h.store.HostStatsInsert(host.ID, stats.CPUPercent, stats.MemPercent, stats.DiskPercent); err != nil {
			log.Printf("history insert host stats: %v", err)
		}

		// 采集运行中的 VM 资源
		vms, err := h.vmManager.List(host.ID)
		if err != nil {
			continue
		}

		for _, v := range vms {
			if v.State != "running" {
				continue
			}
			vmStats, err := h.vmManager.VMStats(host.ID, v.Name)
			if err != nil {
				continue
			}
			h.store.VMStatsInsert(
				host.ID, v.Name,
				vmStats.CPUPercent,
				int64(vmStats.MemRSS),
				int64(vmStats.NetRxBytes),
				int64(vmStats.NetTxBytes),
			)
		}
	}
}
