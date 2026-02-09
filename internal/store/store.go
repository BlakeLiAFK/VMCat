package store

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Store 管理本地数据持久化
type Store struct {
	db *sql.DB
}

// New 创建并初始化数据库
func New() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dbDir := filepath.Join(homeDir, ".vmcat")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dbDir, "vmcat.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Close 关闭数据库
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// migrate 执行数据库迁移
func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS hosts (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		host       TEXT NOT NULL,
		port       INTEGER DEFAULT 22,
		user       TEXT DEFAULT 'root',
		auth_type  TEXT DEFAULT 'key',
		key_path   TEXT DEFAULT '',
		password   TEXT DEFAULT '',
		host_key   TEXT DEFAULT '',
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// 兼容旧库: 添加 host_key 列（已存在则忽略）
	s.db.Exec(`ALTER TABLE hosts ADD COLUMN host_key TEXT DEFAULT ''`)

	// 模板相关表
	if err := s.migrateTemplates(); err != nil {
		return err
	}

	// 兼容旧库: 添加 proxy_addr 列（已存在则忽略）
	s.db.Exec(`ALTER TABLE hosts ADD COLUMN proxy_addr TEXT DEFAULT ''`)

	return nil
}
