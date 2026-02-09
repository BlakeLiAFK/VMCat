package store

import (
	"time"

	"github.com/google/uuid"
)

// Flavor 硬件规格
type Flavor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CPUs      int    `json:"cpus"`
	MemoryMB  int    `json:"memoryMB"`
	DiskGB    int    `json:"diskGB"`
	SortOrder int    `json:"sortOrder"`
	CreatedAt string `json:"createdAt"`
}

// Image OS 模板（按宿主机隔离）
type Image struct {
	ID        string `json:"id"`
	HostID    string `json:"hostId"`
	Name      string `json:"name"`
	BasePath  string `json:"basePath"`
	OSVariant string `json:"osVariant"`
	SortOrder int    `json:"sortOrder"`
	CreatedAt string `json:"createdAt"`
}

// ImageSource 预设镜像源（全局）
type ImageSource struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	OSVariant   string `json:"osVariant"`
	FileName    string `json:"fileName"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
	CreatedAt   string `json:"createdAt"`
}

// Instance 实例记录
type Instance struct {
	ID        int    `json:"id"`
	HostID    string `json:"hostId"`
	VMName    string `json:"vmName"`
	FlavorID  string `json:"flavorId"`
	ImageID   string `json:"imageId"`
	CreatedAt string `json:"createdAt"`
}

// migrateTemplates 创建模板相关表
func (s *Store) migrateTemplates() error {
	schema := `
	CREATE TABLE IF NOT EXISTS flavors (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		cpus       INTEGER NOT NULL DEFAULT 1,
		memory_mb  INTEGER NOT NULL DEFAULT 1024,
		disk_gb    INTEGER NOT NULL DEFAULT 20,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS images (
		id         TEXT PRIMARY KEY,
		host_id    TEXT NOT NULL DEFAULT '',
		name       TEXT NOT NULL,
		base_path  TEXT NOT NULL,
		os_variant TEXT DEFAULT '',
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS image_sources (
		id          TEXT PRIMARY KEY,
		name        TEXT NOT NULL,
		url         TEXT NOT NULL,
		os_variant  TEXT DEFAULT '',
		file_name   TEXT DEFAULT '',
		description TEXT DEFAULT '',
		sort_order  INTEGER DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS instances (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		host_id    TEXT NOT NULL,
		vm_name    TEXT NOT NULL,
		flavor_id  TEXT DEFAULT '',
		image_id   TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// 兼容旧表：为 images 补 host_id 列
	s.db.Exec(`ALTER TABLE images ADD COLUMN host_id TEXT NOT NULL DEFAULT ''`)

	// 插入默认 Flavor（如果表为空）
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM flavors").Scan(&count)
	if count == 0 {
		s.seedDefaultFlavors()
	}

	// 补充缺失的默认镜像源（按 URL 去重）
	s.seedDefaultImageSources()
	return nil
}

// seedDefaultFlavors 插入默认硬件规格
func (s *Store) seedDefaultFlavors() {
	defaults := []Flavor{
		{Name: "mini", CPUs: 1, MemoryMB: 512, DiskGB: 10, SortOrder: 1},
		{Name: "small", CPUs: 1, MemoryMB: 1024, DiskGB: 20, SortOrder: 2},
		{Name: "medium", CPUs: 2, MemoryMB: 4096, DiskGB: 40, SortOrder: 3},
		{Name: "large", CPUs: 4, MemoryMB: 8192, DiskGB: 80, SortOrder: 4},
		{Name: "xlarge", CPUs: 8, MemoryMB: 16384, DiskGB: 160, SortOrder: 5},
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, f := range defaults {
		id := uuid.New().String()
		s.db.Exec(`INSERT INTO flavors (id, name, cpus, memory_mb, disk_gb, sort_order, created_at) VALUES (?,?,?,?,?,?,?)`,
			id, f.Name, f.CPUs, f.MemoryMB, f.DiskGB, f.SortOrder, now)
	}
}

// === Flavor CRUD ===

func (s *Store) FlavorList() ([]Flavor, error) {
	rows, err := s.db.Query(`SELECT id, name, cpus, memory_mb, disk_gb, sort_order, created_at FROM flavors ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Flavor
	for rows.Next() {
		var f Flavor
		if err := rows.Scan(&f.ID, &f.Name, &f.CPUs, &f.MemoryMB, &f.DiskGB, &f.SortOrder, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}

func (s *Store) FlavorAdd(f *Flavor) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`INSERT INTO flavors (id, name, cpus, memory_mb, disk_gb, sort_order, created_at) VALUES (?,?,?,?,?,?,?)`,
		f.ID, f.Name, f.CPUs, f.MemoryMB, f.DiskGB, f.SortOrder, now)
	return err
}

func (s *Store) FlavorUpdate(f *Flavor) error {
	_, err := s.db.Exec(`UPDATE flavors SET name=?, cpus=?, memory_mb=?, disk_gb=?, sort_order=? WHERE id=?`,
		f.Name, f.CPUs, f.MemoryMB, f.DiskGB, f.SortOrder, f.ID)
	return err
}

func (s *Store) FlavorDelete(id string) error {
	_, err := s.db.Exec(`DELETE FROM flavors WHERE id=?`, id)
	return err
}

func (s *Store) FlavorGet(id string) (*Flavor, error) {
	var f Flavor
	err := s.db.QueryRow(`SELECT id, name, cpus, memory_mb, disk_gb, sort_order, created_at FROM flavors WHERE id=?`, id).
		Scan(&f.ID, &f.Name, &f.CPUs, &f.MemoryMB, &f.DiskGB, &f.SortOrder, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// === Image CRUD ===

func (s *Store) ImageList(hostID string) ([]Image, error) {
	rows, err := s.db.Query(`SELECT id, host_id, name, base_path, os_variant, sort_order, created_at FROM images WHERE host_id=? ORDER BY sort_order, created_at`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Image
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.HostID, &img.Name, &img.BasePath, &img.OSVariant, &img.SortOrder, &img.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, img)
	}
	return list, nil
}

func (s *Store) ImageAdd(img *Image) error {
	if img.ID == "" {
		img.ID = uuid.New().String()
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`INSERT INTO images (id, host_id, name, base_path, os_variant, sort_order, created_at) VALUES (?,?,?,?,?,?,?)`,
		img.ID, img.HostID, img.Name, img.BasePath, img.OSVariant, img.SortOrder, now)
	return err
}

func (s *Store) ImageUpdate(img *Image) error {
	_, err := s.db.Exec(`UPDATE images SET name=?, base_path=?, os_variant=?, sort_order=? WHERE id=?`,
		img.Name, img.BasePath, img.OSVariant, img.SortOrder, img.ID)
	return err
}

func (s *Store) ImageDelete(id string) error {
	_, err := s.db.Exec(`DELETE FROM images WHERE id=?`, id)
	return err
}

func (s *Store) ImageGet(id string) (*Image, error) {
	var img Image
	err := s.db.QueryRow(`SELECT id, host_id, name, base_path, os_variant, sort_order, created_at FROM images WHERE id=?`, id).
		Scan(&img.ID, &img.HostID, &img.Name, &img.BasePath, &img.OSVariant, &img.SortOrder, &img.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// === Instance CRUD ===

func (s *Store) InstanceCreate(inst *Instance) (int, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := s.db.Exec(`INSERT INTO instances (host_id, vm_name, flavor_id, image_id, created_at) VALUES (?,?,?,?,?)`,
		inst.HostID, inst.VMName, inst.FlavorID, inst.ImageID, now)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *Store) InstanceGet(id int) (*Instance, error) {
	var inst Instance
	err := s.db.QueryRow(`SELECT id, host_id, vm_name, flavor_id, image_id, created_at FROM instances WHERE id=?`, id).
		Scan(&inst.ID, &inst.HostID, &inst.VMName, &inst.FlavorID, &inst.ImageID, &inst.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

func (s *Store) InstanceList(hostID string) ([]Instance, error) {
	rows, err := s.db.Query(`SELECT id, host_id, vm_name, flavor_id, image_id, created_at FROM instances WHERE host_id=? ORDER BY id`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Instance
	for rows.Next() {
		var inst Instance
		if err := rows.Scan(&inst.ID, &inst.HostID, &inst.VMName, &inst.FlavorID, &inst.ImageID, &inst.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, inst)
	}
	return list, nil
}

func (s *Store) InstanceDelete(id int) error {
	_, err := s.db.Exec(`DELETE FROM instances WHERE id=?`, id)
	return err
}

// InstanceByVMName 按宿主机和 VM 名查找 instance
func (s *Store) InstanceByVMName(hostID, vmName string) (*Instance, error) {
	var inst Instance
	err := s.db.QueryRow(`SELECT id, host_id, vm_name, flavor_id, image_id, created_at FROM instances WHERE host_id=? AND vm_name=?`, hostID, vmName).
		Scan(&inst.ID, &inst.HostID, &inst.VMName, &inst.FlavorID, &inst.ImageID, &inst.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// InstanceUpdateVMName 更新 instance 的 VM 名称（VM 改名时同步）
func (s *Store) InstanceUpdateVMName(id int, newName string) error {
	_, err := s.db.Exec(`UPDATE instances SET vm_name=? WHERE id=?`, newName, id)
	return err
}

// === ImageSource CRUD ===

// seedDefaultImageSources 插入常用云镜像源
func (s *Store) seedDefaultImageSources() {
	defaults := []ImageSource{
		{Name: "Ubuntu 24.04 (Cloud)", URL: "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img", OSVariant: "ubuntu24.04", FileName: "ubuntu-24.04-cloudimg.qcow2", Description: "Ubuntu 24.04 LTS Cloud Image", SortOrder: 1},
		{Name: "Ubuntu 22.04 (Cloud)", URL: "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img", OSVariant: "ubuntu22.04", FileName: "ubuntu-22.04-cloudimg.qcow2", Description: "Ubuntu 22.04 LTS Cloud Image", SortOrder: 2},
		{Name: "Debian 12 (Cloud)", URL: "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2", OSVariant: "debian12", FileName: "debian-12-cloudimg.qcow2", Description: "Debian 12 Bookworm Cloud Image", SortOrder: 3},
		{Name: "CentOS Stream 9", URL: "https://cloud.centos.org/centos/9-stream/x86_64/images/CentOS-Stream-GenericCloud-9-latest.x86_64.qcow2", OSVariant: "centos-stream9", FileName: "centos-stream-9.qcow2", Description: "CentOS Stream 9 Cloud Image", SortOrder: 4},
		{Name: "Rocky Linux 9", URL: "https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2", OSVariant: "rocky9", FileName: "rocky-9-cloudimg.qcow2", Description: "Rocky Linux 9 Cloud Image", SortOrder: 5},
		{Name: "Alpine Linux 3.20 (Cloud)", URL: "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/cloud/nocloud_alpine-3.20.6-x86_64-bios-cloudinit-r0.qcow2", OSVariant: "alpinelinux3.20", FileName: "alpine-3.20-cloud.qcow2", Description: "Alpine Linux 3.20 - Lightweight (~50MB, 128MB+ RAM)", SortOrder: 6},
		{Name: "Cirros 0.6.2 (Test)", URL: "https://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img", OSVariant: "cirros0.6.2", FileName: "cirros-0.6.2.qcow2", Description: "CirrOS 0.6.2 - Minimal test image (~15MB, 64MB+ RAM)", SortOrder: 7},
		{Name: "Debian 12 Minimal (Cloud)", URL: "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-nocloud-amd64.qcow2", OSVariant: "debian12", FileName: "debian-12-nocloud.qcow2", Description: "Debian 12 NoCloud - No cloud-init (~250MB, 256MB+ RAM)", SortOrder: 8},
		{Name: "Ubuntu 24.04 Minimal (Cloud)", URL: "https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-amd64.img", OSVariant: "ubuntu24.04", FileName: "ubuntu-24.04-minimal.qcow2", Description: "Ubuntu 24.04 Minimal - Smaller footprint (~250MB, 256MB+ RAM)", SortOrder: 9},
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	for _, src := range defaults {
		// 按 URL 去重，已存在则跳过
		var exists int
		s.db.QueryRow(`SELECT COUNT(*) FROM image_sources WHERE url = ?`, src.URL).Scan(&exists)
		if exists > 0 {
			continue
		}
		id := uuid.New().String()
		s.db.Exec(`INSERT INTO image_sources (id, name, url, os_variant, file_name, description, sort_order, created_at) VALUES (?,?,?,?,?,?,?,?)`,
			id, src.Name, src.URL, src.OSVariant, src.FileName, src.Description, src.SortOrder, now)
	}
}

func (s *Store) ImageSourceList() ([]ImageSource, error) {
	rows, err := s.db.Query(`SELECT id, name, url, os_variant, file_name, description, sort_order, created_at FROM image_sources ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ImageSource
	for rows.Next() {
		var src ImageSource
		if err := rows.Scan(&src.ID, &src.Name, &src.URL, &src.OSVariant, &src.FileName, &src.Description, &src.SortOrder, &src.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, src)
	}
	return list, nil
}

func (s *Store) ImageSourceAdd(src *ImageSource) error {
	if src.ID == "" {
		src.ID = uuid.New().String()
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := s.db.Exec(`INSERT INTO image_sources (id, name, url, os_variant, file_name, description, sort_order, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		src.ID, src.Name, src.URL, src.OSVariant, src.FileName, src.Description, src.SortOrder, now)
	return err
}

func (s *Store) ImageSourceUpdate(src *ImageSource) error {
	_, err := s.db.Exec(`UPDATE image_sources SET name=?, url=?, os_variant=?, file_name=?, description=?, sort_order=? WHERE id=?`,
		src.Name, src.URL, src.OSVariant, src.FileName, src.Description, src.SortOrder, src.ID)
	return err
}

func (s *Store) ImageSourceDelete(id string) error {
	_, err := s.db.Exec(`DELETE FROM image_sources WHERE id=?`, id)
	return err
}
