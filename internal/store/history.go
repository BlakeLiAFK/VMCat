package store

import "time"

// HostStatsRecord 宿主机资源历史记录
type HostStatsRecord struct {
	ID          int     `json:"id"`
	HostID      string  `json:"hostId"`
	CPUPercent  float64 `json:"cpuPercent"`
	MemPercent  float64 `json:"memPercent"`
	DiskPercent float64 `json:"diskPercent"`
	Timestamp   string  `json:"timestamp"`
}

// VMStatsRecord VM 资源历史记录
type VMStatsRecord struct {
	ID         int     `json:"id"`
	HostID     string  `json:"hostId"`
	VMName     string  `json:"vmName"`
	CPUPercent float64 `json:"cpuPercent"`
	MemUsed    int64   `json:"memUsed"` // bytes
	NetRx      int64   `json:"netRx"`   // bytes
	NetTx      int64   `json:"netTx"`   // bytes
	Timestamp  string  `json:"timestamp"`
}

// migrateHistory 创建历史统计表
func (s *Store) migrateHistory() error {
	schema := `
	CREATE TABLE IF NOT EXISTS host_stats_history (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		host_id      TEXT NOT NULL,
		cpu_percent  REAL,
		mem_percent  REAL,
		disk_percent REAL,
		timestamp    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS vm_stats_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		host_id     TEXT NOT NULL,
		vm_name     TEXT NOT NULL,
		cpu_percent REAL,
		mem_used    INTEGER,
		net_rx      INTEGER,
		net_tx      INTEGER,
		timestamp   DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_host_stats_host_time ON host_stats_history(host_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_vm_stats_host_vm_time ON vm_stats_history(host_id, vm_name, timestamp);
	`
	_, err := s.db.Exec(schema)
	return err
}

// HostStatsInsert 插入宿主机资源记录
func (s *Store) HostStatsInsert(hostID string, cpu, mem, disk float64) error {
	_, err := s.db.Exec(`
		INSERT INTO host_stats_history (host_id, cpu_percent, mem_percent, disk_percent)
		VALUES (?, ?, ?, ?)
	`, hostID, cpu, mem, disk)
	return err
}

// VMStatsInsert 插入 VM 资源记录
func (s *Store) VMStatsInsert(hostID, vmName string, cpu float64, memUsed, netRx, netTx int64) error {
	_, err := s.db.Exec(`
		INSERT INTO vm_stats_history (host_id, vm_name, cpu_percent, mem_used, net_rx, net_tx)
		VALUES (?, ?, ?, ?, ?, ?)
	`, hostID, vmName, cpu, memUsed, netRx, netTx)
	return err
}

// HostStatsHistory 获取宿主机资源历史
func (s *Store) HostStatsHistory(hostID string, hours int) ([]HostStatsRecord, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02 15:04:05")
	rows, err := s.db.Query(`
		SELECT id, host_id, cpu_percent, mem_percent, disk_percent, timestamp
		FROM host_stats_history
		WHERE host_id = ? AND timestamp > ?
		ORDER BY timestamp
	`, hostID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []HostStatsRecord
	for rows.Next() {
		var r HostStatsRecord
		if err := rows.Scan(&r.ID, &r.HostID, &r.CPUPercent, &r.MemPercent, &r.DiskPercent, &r.Timestamp); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// VMStatsHistory 获取 VM 资源历史
func (s *Store) VMStatsHistory(hostID, vmName string, hours int) ([]VMStatsRecord, error) {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02 15:04:05")
	rows, err := s.db.Query(`
		SELECT id, host_id, vm_name, cpu_percent, mem_used, net_rx, net_tx, timestamp
		FROM vm_stats_history
		WHERE host_id = ? AND vm_name = ? AND timestamp > ?
		ORDER BY timestamp
	`, hostID, vmName, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []VMStatsRecord
	for rows.Next() {
		var r VMStatsRecord
		if err := rows.Scan(&r.ID, &r.HostID, &r.VMName, &r.CPUPercent, &r.MemUsed, &r.NetRx, &r.NetTx, &r.Timestamp); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// StatsCleanup 清理超过指定小时的历史数据
func (s *Store) StatsCleanup(hours int) error {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02 15:04:05")
	s.db.Exec(`DELETE FROM host_stats_history WHERE timestamp < ?`, cutoff)
	s.db.Exec(`DELETE FROM vm_stats_history WHERE timestamp < ?`, cutoff)
	return nil
}
