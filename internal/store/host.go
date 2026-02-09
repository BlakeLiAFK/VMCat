package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Host 宿主机配置
type Host struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	AuthType  string `json:"authType"`  // key | password
	KeyPath   string `json:"keyPath"`
	Password  string `json:"password"`
	SortOrder int    `json:"sortOrder"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// HostList 获取所有宿主机
func (s *Store) HostList() ([]Host, error) {
	rows, err := s.db.Query(`
		SELECT id, name, host, port, user, auth_type, key_path, password, sort_order, created_at, updated_at
		FROM hosts ORDER BY sort_order, created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		var h Host
		if err := rows.Scan(&h.ID, &h.Name, &h.Host, &h.Port, &h.User, &h.AuthType,
			&h.KeyPath, &h.Password, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		// 不返回密码给前端
		h.Password = ""
		hosts = append(hosts, h)
	}
	return hosts, nil
}

// HostGet 获取单个宿主机
func (s *Store) HostGet(id string) (*Host, error) {
	var h Host
	err := s.db.QueryRow(`
		SELECT id, name, host, port, user, auth_type, key_path, password, sort_order, created_at, updated_at
		FROM hosts WHERE id = ?
	`, id).Scan(&h.ID, &h.Name, &h.Host, &h.Port, &h.User, &h.AuthType,
		&h.KeyPath, &h.Password, &h.SortOrder, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// HostAdd 添加宿主机
func (s *Store) HostAdd(h *Host) error {
	if h.ID == "" {
		h.ID = uuid.New().String()
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`
		INSERT INTO hosts (id, name, host, port, user, auth_type, key_path, password, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, h.ID, h.Name, h.Host, h.Port, h.User, h.AuthType, h.KeyPath, h.Password, h.SortOrder, now, now)
	return err
}

// HostUpdate 更新宿主机
func (s *Store) HostUpdate(h *Host) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	// 如果密码为空，不更新密码字段
	if h.Password == "" {
		_, err := s.db.Exec(`
			UPDATE hosts SET name=?, host=?, port=?, user=?, auth_type=?, key_path=?, sort_order=?, updated_at=?
			WHERE id=?
		`, h.Name, h.Host, h.Port, h.User, h.AuthType, h.KeyPath, h.SortOrder, now, h.ID)
		return err
	}
	_, err := s.db.Exec(`
		UPDATE hosts SET name=?, host=?, port=?, user=?, auth_type=?, key_path=?, password=?, sort_order=?, updated_at=?
		WHERE id=?
	`, h.Name, h.Host, h.Port, h.User, h.AuthType, h.KeyPath, h.Password, h.SortOrder, now, h.ID)
	return err
}

// HostDelete 删除宿主机
func (s *Store) HostDelete(id string) error {
	_, err := s.db.Exec(`DELETE FROM hosts WHERE id = ?`, id)
	return err
}

// HostExportJSON 导出所有宿主机为 JSON（不含密码）
func (s *Store) HostExportJSON() (string, error) {
	hosts, err := s.HostList()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// HostImportJSON 从 JSON 导入宿主机（跳过已存在的）
func (s *Store) HostImportJSON(jsonStr string) (int, error) {
	var hosts []Host
	if err := json.Unmarshal([]byte(jsonStr), &hosts); err != nil {
		return 0, err
	}

	imported := 0
	for _, h := range hosts {
		existing, _ := s.HostGet(h.ID)
		if existing != nil {
			continue
		}
		if err := s.HostAdd(&h); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}
