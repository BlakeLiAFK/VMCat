package store

// AuditRecord 审计日志记录
type AuditRecord struct {
	ID        int    `json:"id"`
	HostID    string `json:"hostId"`
	VMName    string `json:"vmName"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	Timestamp string `json:"timestamp"`
}

// migrateAudit 创建审计日志表
func (s *Store) migrateAudit() error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		host_id   TEXT,
		vm_name   TEXT,
		action    TEXT NOT NULL,
		detail    TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_audit_host ON audit_log(host_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action, timestamp);
	`
	_, err := s.db.Exec(schema)
	return err
}

// AuditInsert 插入审计日志
func (s *Store) AuditInsert(hostID, vmName, action, detail string) error {
	_, err := s.db.Exec(`
		INSERT INTO audit_log (host_id, vm_name, action, detail)
		VALUES (?, ?, ?, ?)
	`, hostID, vmName, action, detail)
	return err
}

// AuditList 获取指定宿主机的审计日志
func (s *Store) AuditList(hostID string, limit int) ([]AuditRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT id, host_id, vm_name, action, detail, timestamp
		FROM audit_log
		WHERE host_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, hostID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AuditRecord
	for rows.Next() {
		var r AuditRecord
		if err := rows.Scan(&r.ID, &r.HostID, &r.VMName, &r.Action, &r.Detail, &r.Timestamp); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// AuditListAll 获取全部审计日志
func (s *Store) AuditListAll(limit int) ([]AuditRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.Query(`
		SELECT id, host_id, vm_name, action, detail, timestamp
		FROM audit_log
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AuditRecord
	for rows.Next() {
		var r AuditRecord
		if err := rows.Scan(&r.ID, &r.HostID, &r.VMName, &r.Action, &r.Detail, &r.Timestamp); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
