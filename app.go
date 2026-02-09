package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"vmcat/internal/monitor"
	internalssh "vmcat/internal/ssh"
	"vmcat/internal/store"
	"vmcat/internal/terminal"
	"vmcat/internal/vm"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

// App Wails 绑定层
type App struct {
	ctx              context.Context
	store            *store.Store
	sshPool          *internalssh.Pool
	vmManager        *vm.Manager
	monitor          *monitor.Collector
	historyCollector *monitor.HistoryCollector
	termSrv          *terminal.Server
	forceQuit        bool // 真正退出标志，由托盘"退出"菜单设置
	importMu         sync.Mutex
	importTasks      map[string]*importTask // 活跃的导入任务
}

// importTask 镜像导入任务
type importTask struct {
	ID        string `json:"id"`
	HostID    string `json:"hostId"`
	Status    string `json:"status"` // downloading, uploading, done, error
	Percent   int    `json:"percent"`
	TotalSize int64  `json:"totalSize"`
	Current   int64  `json:"current"`
	Error     string `json:"error"`
}

func NewApp() *App {
	pool := internalssh.NewPool()
	return &App{
		sshPool:   pool,
		vmManager: vm.NewManager(pool),
		monitor:   monitor.NewCollector(pool),
		termSrv:   terminal.NewServer(pool),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	s, err := store.New()
	if err != nil {
		log.Printf("init store: %v", err)
		return
	}
	a.store = s

	// 迁移旧的明文密码为加密格式
	s.MigrateEncryptPasswords()

	// 启动终端 WebSocket 服务
	if err := a.termSrv.Start(); err != nil {
		log.Printf("start terminal server: %v", err)
	}

	// 启动资源历史采集器
	a.historyCollector = monitor.NewHistoryCollector(a.sshPool, a.store, a.monitor, a.vmManager)
	a.historyCollector.Start()
}

// beforeClose 拦截窗口关闭事件，隐藏到系统托盘
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.forceQuit {
		return false
	}
	wailsRuntime.WindowHide(ctx)
	return true
}

func (a *App) shutdown(ctx context.Context) {
	if a.historyCollector != nil {
		a.historyCollector.Stop()
	}
	a.termSrv.Close()
	a.sshPool.CloseAll()
	if a.store != nil {
		a.store.Close()
	}
}

// audit 记录审计日志（内部辅助方法）
func (a *App) audit(hostID, vmName, action, detail string) {
	if a.store != nil {
		a.store.AuditInsert(hostID, vmName, action, detail)
	}
}

// === 宿主机管理 ===

// HostList 获取宿主机列表
func (a *App) HostList() ([]store.Host, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	hosts, err := a.store.HostList()
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		if a.sshPool.IsConnected(hosts[i].ID) {
			hosts[i].SortOrder = -1
		}
	}
	return hosts, nil
}

// HostAdd 添加宿主机
func (a *App) HostAdd(h store.Host) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.HostAdd(&h)
}

// HostUpdate 更新宿主机
func (a *App) HostUpdate(h store.Host) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.HostUpdate(&h)
}

// HostDelete 删除宿主机
func (a *App) HostDelete(id string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	a.sshPool.Disconnect(id)
	return a.store.HostDelete(id)
}

// HostConnect 连接到宿主机
func (a *App) HostConnect(id string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	h, err := a.store.HostGet(id)
	if err != nil {
		return fmt.Errorf("host not found: %w", err)
	}

	cfg := &internalssh.Config{
		Host:      h.Host,
		Port:      h.Port,
		User:      h.User,
		AuthType:  h.AuthType,
		KeyPath:   h.KeyPath,
		Password:  h.Password,
		HostKey:   h.HostKey,
		ProxyAddr: h.ProxyAddr,
	}

	client, err := a.sshPool.Connect(id, cfg)
	if err != nil {
		return err
	}

	// 首次连接（无已知密钥），存储服务端公钥
	if h.HostKey == "" {
		if key := client.ConnectedHostKey(); key != "" {
			a.store.HostUpdateHostKey(id, key)
		}
	}

	return nil
}

// HostDisconnect 断开宿主机
func (a *App) HostDisconnect(id string) {
	a.sshPool.Disconnect(id)
}

// HostTest 测试 SSH 连接，返回 "hostname (fingerprint)" 格式的信息
func (a *App) HostTest(h store.Host) (string, error) {
	cfg := &internalssh.Config{
		Host:      h.Host,
		Port:      h.Port,
		User:      h.User,
		AuthType:  h.AuthType,
		KeyPath:   h.KeyPath,
		Password:  h.Password,
		ProxyAddr: h.ProxyAddr,
	}

	client := internalssh.NewClient(cfg)
	if err := client.Connect(); err != nil {
		return "", err
	}

	output, err := client.Execute("hostname")
	if err != nil {
		client.Close()
		return "", fmt.Errorf("execute test command: %w", err)
	}

	// 获取指纹信息
	fingerprint := ""
	if key := client.ConnectedHostKey(); key != "" {
		raw, _ := base64.StdEncoding.DecodeString(key)
		if len(raw) > 0 {
			h := sha256.Sum256(raw)
			fingerprint = "SHA256:" + base64.StdEncoding.EncodeToString(h[:])
		}
	}

	client.Close()
	log.Printf("host test ok: %s", output)

	if fingerprint != "" {
		return fmt.Sprintf("%s (%s)", output, fingerprint), nil
	}
	return output, nil
}

// HostResetHostKey 重置宿主机的 SSH 主机密钥
func (a *App) HostResetHostKey(id string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.HostUpdateHostKey(id, "")
}

// HostGetFingerprint 获取已存储的主机密钥指纹
func (a *App) HostGetFingerprint(id string) (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("store not initialized")
	}
	h, err := a.store.HostGet(id)
	if err != nil {
		return "", err
	}
	if h.HostKey == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(h.HostKey)
	if err != nil || len(raw) == 0 {
		return "", nil
	}
	hash := sha256.Sum256(raw)
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:]), nil
}

// HostIsConnected 检查宿主机连接状态
func (a *App) HostIsConnected(id string) bool {
	return a.sshPool.IsConnected(id)
}

// HostResourceStats 获取宿主机资源统计
func (a *App) HostResourceStats(hostID string) (*monitor.HostStats, error) {
	return a.monitor.Collect(hostID)
}

// HostExportJSON 导出宿主机配置
func (a *App) HostExportJSON() (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("store not initialized")
	}
	return a.store.HostExportJSON()
}

// HostImportJSON 导入宿主机配置
func (a *App) HostImportJSON(jsonStr string) (int, error) {
	if a.store == nil {
		return 0, fmt.Errorf("store not initialized")
	}
	return a.store.HostImportJSON(jsonStr)
}

// === VM 管理 ===

// VMList 获取虚拟机列表
func (a *App) VMList(hostID string) ([]vm.VM, error) {
	return a.vmManager.List(hostID)
}

// VMGet 获取虚拟机详情
func (a *App) VMGet(hostID, vmName string) (*vm.VMDetail, error) {
	return a.vmManager.Get(hostID, vmName)
}

// VMStart 启动虚拟机
func (a *App) VMStart(hostID, vmName string) error {
	err := a.vmManager.Start(hostID, vmName)
	if err == nil {
		a.audit(hostID, vmName, "vm.start", "")
	}
	return err
}

// VMShutdown 关闭虚拟机
func (a *App) VMShutdown(hostID, vmName string) error {
	err := a.vmManager.Shutdown(hostID, vmName)
	if err == nil {
		a.audit(hostID, vmName, "vm.shutdown", "")
	}
	return err
}

// VMDestroy 强制关闭虚拟机
func (a *App) VMDestroy(hostID, vmName string) error {
	err := a.vmManager.Destroy(hostID, vmName)
	if err == nil {
		a.audit(hostID, vmName, "vm.destroy", "")
	}
	return err
}

// VMReboot 重启虚拟机
func (a *App) VMReboot(hostID, vmName string) error {
	err := a.vmManager.Reboot(hostID, vmName)
	if err == nil {
		a.audit(hostID, vmName, "vm.reboot", "")
	}
	return err
}

// VMSuspend 暂停虚拟机
func (a *App) VMSuspend(hostID, vmName string) error {
	return a.vmManager.Suspend(hostID, vmName)
}

// VMResume 恢复虚拟机
func (a *App) VMResume(hostID, vmName string) error {
	return a.vmManager.Resume(hostID, vmName)
}

// VMDelete 删除虚拟机
func (a *App) VMDelete(hostID, vmName string, removeStorage bool) error {
	err := a.vmManager.Delete(hostID, vmName, removeStorage)
	if err == nil {
		detail := ""
		if removeStorage {
			detail = "含存储"
		}
		a.audit(hostID, vmName, "vm.delete", detail)
	}
	return err
}

// VMRename 重命名虚拟机
func (a *App) VMRename(hostID, oldName, newName string) error {
	return a.vmManager.Rename(hostID, oldName, newName)
}

// VMSetVCPUs 设置 CPU 数量
func (a *App) VMSetVCPUs(hostID, vmName string, count int) error {
	return a.vmManager.SetVCPUs(hostID, vmName, count)
}

// VMSetMemory 设置内存大小 (MB)
func (a *App) VMSetMemory(hostID, vmName string, sizeMB int) error {
	return a.vmManager.SetMemory(hostID, vmName, sizeMB)
}

// VMSetAutostart 设置自动启动
func (a *App) VMSetAutostart(hostID, vmName string, enabled bool) error {
	return a.vmManager.SetAutostart(hostID, vmName, enabled)
}

// VMClone 克隆虚拟机
func (a *App) VMClone(hostID, srcName, newName string) error {
	err := a.vmManager.Clone(hostID, srcName, newName)
	if err == nil {
		a.audit(hostID, newName, "vm.clone", fmt.Sprintf("from %s", srcName))
	}
	return err
}

// VMGetXML 获取 VM XML 配置
func (a *App) VMGetXML(hostID, vmName string) (string, error) {
	return a.vmManager.GetXML(hostID, vmName)
}

// VMDefineXML 用 XML 定义/更新 VM
func (a *App) VMDefineXML(hostID, xmlContent string) error {
	return a.vmManager.DefineXML(hostID, xmlContent)
}

// VMCreate 创建虚拟机
func (a *App) VMCreate(hostID string, params vm.VMCreateParams) error {
	err := a.vmManager.Create(hostID, params)
	if err == nil {
		a.audit(hostID, params.Name, "vm.create", "")
	}
	return err
}

// VMStats 获取 VM 实时资源统计
func (a *App) VMStats(hostID, vmName string) (*vm.VMResourceStats, error) {
	return a.vmManager.VMStats(hostID, vmName)
}

// === 硬件管理 ===

// VMAttachDisk 添加磁盘
func (a *App) VMAttachDisk(hostID, vmName string, params vm.DiskAttachParams) error {
	return a.vmManager.AttachDisk(hostID, vmName, params)
}

// VMDetachDisk 移除磁盘
func (a *App) VMDetachDisk(hostID, vmName, target string) error {
	return a.vmManager.DetachDisk(hostID, vmName, target)
}

// VMAttachInterface 添加网卡
func (a *App) VMAttachInterface(hostID, vmName string, params vm.NICAttachParams) error {
	return a.vmManager.AttachInterface(hostID, vmName, params)
}

// VMDetachInterface 移除网卡
func (a *App) VMDetachInterface(hostID, vmName, macAddr string) error {
	return a.vmManager.DetachInterface(hostID, vmName, macAddr)
}

// VMChangeMedia 挂载光驱
func (a *App) VMChangeMedia(hostID, vmName, target, source string) error {
	return a.vmManager.ChangeMedia(hostID, vmName, target, source)
}

// VMEjectMedia 弹出光驱
func (a *App) VMEjectMedia(hostID, vmName, target string) error {
	return a.vmManager.EjectMedia(hostID, vmName, target)
}

// VMResizeDisk 磁盘扩容
func (a *App) VMResizeDisk(hostID, diskPath string, newSizeGB int) error {
	return a.vmManager.ResizeDisk(hostID, diskPath, newSizeGB)
}

// VMSetGraphics 设置 VNC 显示
func (a *App) VMSetGraphics(hostID, vmName string, enabled bool) error {
	return a.vmManager.SetGraphics(hostID, vmName, enabled)
}

// === 存储管理 ===

// PoolList 获取存储池列表
func (a *App) PoolList(hostID string) ([]vm.StoragePool, error) {
	return a.vmManager.PoolList(hostID)
}

// VolList 获取卷列表
func (a *App) VolList(hostID, poolName string) ([]vm.Volume, error) {
	return a.vmManager.VolList(hostID, poolName)
}

// CreateVolume 创建卷
func (a *App) CreateVolume(hostID, poolName, volName string, sizeGB int, format string) (string, error) {
	return a.vmManager.CreateVolume(hostID, poolName, volName, sizeGB, format)
}

// DeleteVolume 删除卷
func (a *App) DeleteVolume(hostID, poolName, volName string) error {
	return a.vmManager.DeleteVolume(hostID, poolName, volName)
}

// === 存储池管理 ===

// PoolStart 启动存储池
func (a *App) PoolStart(hostID, poolName string) error {
	return a.vmManager.PoolStart(hostID, poolName)
}

// PoolStop 停止存储池
func (a *App) PoolStop(hostID, poolName string) error {
	return a.vmManager.PoolStop(hostID, poolName)
}

// PoolAutostart 设置存储池自动启动
func (a *App) PoolAutostart(hostID, poolName string, enabled bool) error {
	return a.vmManager.PoolAutostart(hostID, poolName, enabled)
}

// === 网络管理 ===

// NetworkList 获取网络列表
func (a *App) NetworkList(hostID string) ([]vm.Network, error) {
	return a.vmManager.NetworkList(hostID)
}

// BridgeList 获取网桥列表
func (a *App) BridgeList(hostID string) ([]string, error) {
	return a.vmManager.BridgeList(hostID)
}

// NetworkStart 启动虚拟网络
func (a *App) NetworkStart(hostID, netName string) error {
	return a.vmManager.NetworkStart(hostID, netName)
}

// NetworkStop 停止虚拟网络
func (a *App) NetworkStop(hostID, netName string) error {
	return a.vmManager.NetworkStop(hostID, netName)
}

// NetworkAutostart 设置虚拟网络自动启动
func (a *App) NetworkAutostart(hostID, netName string, enabled bool) error {
	return a.vmManager.NetworkAutostart(hostID, netName, enabled)
}

// === NAT 端口转发 ===

// NATRuleList 列出 NAT 端口转发规则
func (a *App) NATRuleList(hostID string) ([]vm.NATRule, error) {
	return a.vmManager.NATRuleList(hostID)
}

// NATRuleAdd 添加 NAT 端口转发规则
func (a *App) NATRuleAdd(hostID, proto, hostPort, vmIP, vmPort, comment string) error {
	return a.vmManager.NATRuleAdd(hostID, proto, hostPort, vmIP, vmPort, comment)
}

// NATRuleDelete 删除 NAT 端口转发规则
func (a *App) NATRuleDelete(hostID, proto, hostPort, vmIP, vmPort string) error {
	return a.vmManager.NATRuleDelete(hostID, proto, hostPort, vmIP, vmPort)
}

// === ISO 管理 ===

// ISOList 列出 ISO 镜像
func (a *App) ISOList(hostID string) ([]vm.ISOFile, error) {
	var paths []string
	if a.store != nil {
		if val, _ := a.store.SettingGet("iso_search_paths"); val != "" {
			for _, p := range strings.Split(val, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					paths = append(paths, p)
				}
			}
		}
	}
	return a.vmManager.ISOList(hostID, paths)
}

// OSVariantList 获取 OS 变体列表
func (a *App) OSVariantList(hostID string) ([]string, error) {
	return a.vmManager.OSVariantList(hostID)
}

// === 快照管理 ===

// SnapshotList 获取快照列表
func (a *App) SnapshotList(hostID, vmName string) ([]vm.Snapshot, error) {
	return a.vmManager.SnapshotList(hostID, vmName)
}

// SnapshotCreate 创建快照
func (a *App) SnapshotCreate(hostID, vmName, snapName string) error {
	return a.vmManager.SnapshotCreate(hostID, vmName, snapName)
}

// SnapshotDelete 删除快照
func (a *App) SnapshotDelete(hostID, vmName, snapName string) error {
	return a.vmManager.SnapshotDelete(hostID, vmName, snapName)
}

// SnapshotRevert 恢复快照
func (a *App) SnapshotRevert(hostID, vmName, snapName string) error {
	return a.vmManager.SnapshotRevert(hostID, vmName, snapName)
}

// === 终端 ===

// TerminalPort 获取终端 WebSocket 服务端口
func (a *App) TerminalPort() int {
	return a.termSrv.Port()
}

// === 设置 ===

// SettingGet 获取设置
func (a *App) SettingGet(key string) (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("store not initialized")
	}
	return a.store.SettingGet(key)
}

// SettingSet 保存设置
func (a *App) SettingSet(key, value string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.SettingSet(key, value)
}

// === 模板管理 ===

// FlavorList 获取硬件规格列表
func (a *App) FlavorList() ([]store.Flavor, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.FlavorList()
}

// FlavorAdd 添加硬件规格
func (a *App) FlavorAdd(f store.Flavor) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.FlavorAdd(&f)
}

// FlavorUpdate 更新硬件规格
func (a *App) FlavorUpdate(f store.Flavor) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.FlavorUpdate(&f)
}

// FlavorDelete 删除硬件规格
func (a *App) FlavorDelete(id string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.FlavorDelete(id)
}

// ImageList 获取宿主机的 OS 模板列表
func (a *App) ImageList(hostID string) ([]store.Image, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.ImageList(hostID)
}

// ImageAdd 添加宿主机的 OS 模板
func (a *App) ImageAdd(hostID string, img store.Image) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	img.HostID = hostID
	return a.store.ImageAdd(&img)
}

// ImageUpdate 更新 OS 模板
func (a *App) ImageUpdate(img store.Image) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.ImageUpdate(&img)
}

// ImageDelete 删除 OS 模板
func (a *App) ImageDelete(id string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.ImageDelete(id)
}

// VMCreateFromTemplate 基于模板快速创建 VM
func (a *App) VMCreateFromTemplate(hostID, vmName, flavorID, imageID, netType, netName, rootPassword, sshPubKey string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}

	flavor, err := a.store.FlavorGet(flavorID)
	if err != nil {
		return fmt.Errorf("flavor not found: %w", err)
	}
	image, err := a.store.ImageGet(imageID)
	if err != nil {
		return fmt.Errorf("image not found: %w", err)
	}

	// 读取 instance_root 配置
	instanceRoot, _ := a.store.SettingGet("instance_root")
	if instanceRoot == "" {
		instanceRoot = "/var/lib/libvirt/instances"
	}

	// 创建 instance 记录，获取自增 ID
	inst := &store.Instance{
		HostID:   hostID,
		VMName:   vmName,
		FlavorID: flavorID,
		ImageID:  imageID,
	}
	instanceID, err := a.store.InstanceCreate(inst)
	if err != nil {
		return fmt.Errorf("create instance record: %w", err)
	}

	// 调用 VM 创建
	params := &vm.TemplateCreateParams{
		VMName:       vmName,
		InstanceID:   instanceID,
		InstanceRoot: instanceRoot,
		CPUs:         flavor.CPUs,
		MemoryMB:     flavor.MemoryMB,
		DiskGB:       flavor.DiskGB,
		BasePath:     image.BasePath,
		OSVariant:    image.OSVariant,
		NetType:      netType,
		NetName:      netName,
		RootPassword: rootPassword,
		SSHPubKey:    sshPubKey,
	}

	if err := a.vmManager.CreateFromTemplate(hostID, params); err != nil {
		// 创建失败，删除 instance 记录
		a.store.InstanceDelete(instanceID)
		return err
	}

	return nil
}

// InstanceISOList 获取 instance 专属 ISO 列表
func (a *App) InstanceISOList(hostID string, instanceID int) ([]vm.ISOFile, error) {
	instanceRoot, _ := a.store.SettingGet("instance_root")
	if instanceRoot == "" {
		instanceRoot = "/var/lib/libvirt/instances"
	}
	return a.vmManager.InstanceISOList(hostID, instanceRoot, instanceID)
}

// InstanceList 获取宿主机的 instance 列表
func (a *App) InstanceList(hostID string) ([]store.Instance, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.InstanceList(hostID)
}

// InstanceByVMName 按 VM 名查找 instance
func (a *App) InstanceByVMName(hostID, vmName string) (*store.Instance, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.InstanceByVMName(hostID, vmName)
}

// === 工具 ===

// AppVersion 获取应用版本号（从 wails.json 读取）
func (a *App) AppVersion() string {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(wailsJSON, &cfg); err != nil {
		return "unknown"
	}
	return cfg.Info.ProductVersion
}

// === 资源历史 ===

// HostStatsHistory 获取宿主机资源历史
func (a *App) HostStatsHistory(hostID string, hours int) ([]store.HostStatsRecord, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if hours <= 0 {
		hours = 24
	}
	return a.store.HostStatsHistory(hostID, hours)
}

// VMStatsHistory 获取 VM 资源历史
func (a *App) VMStatsHistory(hostID, vmName string, hours int) ([]store.VMStatsRecord, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if hours <= 0 {
		hours = 24
	}
	return a.store.VMStatsHistory(hostID, vmName, hours)
}

// === 审计日志 ===

// AuditList 获取指定宿主机的审计日志
func (a *App) AuditList(hostID string, limit int) ([]store.AuditRecord, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.AuditList(hostID, limit)
}

// AuditListAll 获取全部审计日志
func (a *App) AuditListAll(limit int) ([]store.AuditRecord, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.AuditListAll(limit)
}

// === VM 备注 ===

// VMNoteGet 获取 VM 备注
func (a *App) VMNoteGet(hostID, vmName string) (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("store not initialized")
	}
	key := fmt.Sprintf("vm_note:%s:%s", hostID, vmName)
	return a.store.SettingGet(key)
}

// VMNoteSet 设置 VM 备注
func (a *App) VMNoteSet(hostID, vmName, note string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	key := fmt.Sprintf("vm_note:%s:%s", hostID, vmName)
	return a.store.SettingSet(key, note)
}

// === Cloud-Init ===

// VMGenerateCloudInit 生成 cloud-init ISO
func (a *App) VMGenerateCloudInit(hostID, outputPath string, cfg vm.CloudInitConfig) error {
	err := a.vmManager.GenerateCloudInitISO(hostID, outputPath, cfg)
	if err == nil {
		a.audit(hostID, cfg.Hostname, "cloudinit.generate", outputPath)
	}
	return err
}

// === VM 迁移 ===

// VMMigrate 在线迁移 VM
func (a *App) VMMigrate(srcHostID, vmName, dstHostID string) error {
	err := a.vmManager.Migrate(srcHostID, vmName, dstHostID)
	if err == nil {
		a.audit(srcHostID, vmName, "vm.migrate", fmt.Sprintf("to %s", dstHostID))
	}
	return err
}

// VMMigrateOffline 离线迁移 VM (通过客户端中继，适用于网络隔离场景)
func (a *App) VMMigrateOffline(srcHostID, vmName, dstHostID string) error {
	err := a.vmManager.MigrateOffline(srcHostID, vmName, dstHostID, func(step, detail string) {
		runtime.EventsEmit(a.ctx, "migrate:progress", map[string]string{
			"step":   step,
			"detail": detail,
		})
	})
	if err == nil {
		a.audit(srcHostID, vmName, "vm.migrate_offline", fmt.Sprintf("to %s", dstHostID))
	}
	return err
}

// HostCheckTools 检测宿主机上的工具安装情况
func (a *App) HostCheckTools(id string) (map[string]string, error) {
	client, err := a.sshPool.Get(id)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	out, err := client.Execute("virsh --version 2>/dev/null")
	if err != nil {
		result["virsh"] = ""
	} else {
		result["virsh"] = out
	}
	return result, nil
}

// === 宿主机镜像文件管理 ===

// HostImageFile 宿主机上的镜像文件
type HostImageFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size string `json:"size"`
}

// HostImageScan 扫描宿主机上的镜像文件
func (a *App) HostImageScan(hostID string) ([]HostImageFile, error) {
	client, err := a.sshPool.Get(hostID)
	if err != nil {
		return nil, err
	}
	// 扫描常见镜像目录
	cmd := `find /var/lib/libvirt/images /root /home /opt -maxdepth 3 \( -name '*.qcow2' -o -name '*.img' -o -name '*.raw' -o -name '*.vmdk' \) -type f -printf '%s %p\n' 2>/dev/null | head -200`
	output, err := client.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("scan images: %w", err)
	}
	var files []HostImageFile
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		sizeBytes := parts[0]
		path := parts[1]
		// 格式化文件大小
		var sizeStr string
		if n, err := strconv.ParseInt(sizeBytes, 10, 64); err == nil {
			if n < 1024*1024 {
				sizeStr = fmt.Sprintf("%d KB", n/1024)
			} else if n < 1024*1024*1024 {
				sizeStr = fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
			} else {
				sizeStr = fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
			}
		} else {
			sizeStr = sizeBytes
		}
		// 提取文件名
		name := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		}
		files = append(files, HostImageFile{Name: name, Path: path, Size: sizeStr})
	}
	return files, nil
}

// HostImageDelete 删除宿主机上的镜像文件
func (a *App) HostImageDelete(hostID, path string) error {
	if path == "" || path == "/" {
		return fmt.Errorf("invalid path")
	}
	client, err := a.sshPool.Get(hostID)
	if err != nil {
		return err
	}
	output, err := client.Execute(fmt.Sprintf("rm -f %s", path))
	if err != nil {
		return fmt.Errorf("delete failed: %s", output)
	}
	return nil
}

// === Libvirt 安装脚本 ===

// LibvirtSetupScript 安装脚本定义
type LibvirtSetupScript struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Distros     string `json:"distros"`
	Script      string `json:"script"`
}

// HostDetectDistro 检测远程主机发行版
func (a *App) HostDetectDistro(hostID string) (string, error) {
	client, err := a.sshPool.Get(hostID)
	if err != nil {
		return "", err
	}
	out, err := client.Execute("cat /etc/os-release 2>/dev/null | grep ^ID= | head -1 | cut -d= -f2 | tr -d '\"'")
	if err != nil {
		return "unknown", nil
	}
	return strings.TrimSpace(out), nil
}

// LibvirtSetupScriptList 获取可用的 libvirt 安装脚本列表
func (a *App) LibvirtSetupScriptList() []LibvirtSetupScript {
	return []LibvirtSetupScript{
		{
			ID:          "debian-ubuntu",
			Name:        "Debian / Ubuntu",
			Description: "apt-based install for Debian, Ubuntu and derivatives",
			Distros:     "debian,ubuntu,linuxmint,pop",
			Script: `#!/bin/bash
set -e

echo "==> Checking virtualization support..."
if [ $(egrep -c '(vmx|svm)' /proc/cpuinfo) -eq 0 ]; then
    echo "ERROR: CPU does not support virtualization (VT-x/AMD-V) or it is disabled in BIOS."
    exit 1
fi

echo "==> Updating package lists..."
sudo apt-get update -qq

echo "==> Installing KVM, QEMU, Libvirt and tools..."
sudo apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virtinst genisoimage wget curl

echo "==> Adding current user to libvirt and kvm groups..."
sudo usermod -aG libvirt $USER 2>/dev/null || true
sudo usermod -aG kvm $USER 2>/dev/null || true

echo "==> Enabling and starting libvirtd..."
sudo systemctl enable --now libvirtd

echo "==> Verifying installation..."
if systemctl is-active --quiet libvirtd; then
    echo "SUCCESS: libvirtd is running."
    echo "virsh version: $(virsh --version 2>/dev/null || echo 'N/A')"
    echo "NOTE: Log out and log back in for group permissions to take effect."
else
    echo "ERROR: libvirtd failed to start. Check logs with: journalctl -u libvirtd"
    exit 1
fi
`,
		},
		{
			ID:          "rhel-centos-rocky",
			Name:        "RHEL / CentOS / Rocky / AlmaLinux",
			Description: "dnf/yum-based install for RHEL family",
			Distros:     "rhel,centos,rocky,almalinux,fedora,ol",
			Script: `#!/bin/bash
set -e

echo "==> Checking virtualization support..."
if [ $(egrep -c '(vmx|svm)' /proc/cpuinfo) -eq 0 ]; then
    echo "ERROR: CPU does not support virtualization (VT-x/AMD-V) or it is disabled in BIOS."
    exit 1
fi

echo "==> Installing KVM, QEMU, Libvirt and tools..."
if command -v dnf &>/dev/null; then
    sudo dnf install -y qemu-kvm libvirt libvirt-client virt-install bridge-utils genisoimage wget curl
else
    sudo yum install -y qemu-kvm libvirt libvirt-client virt-install bridge-utils genisoimage wget curl
fi

echo "==> Adding current user to libvirt group..."
sudo usermod -aG libvirt $USER 2>/dev/null || true

echo "==> Enabling and starting libvirtd..."
sudo systemctl enable --now libvirtd

echo "==> Verifying installation..."
if systemctl is-active --quiet libvirtd; then
    echo "SUCCESS: libvirtd is running."
    echo "virsh version: $(virsh --version 2>/dev/null || echo 'N/A')"
    echo "NOTE: Log out and log back in for group permissions to take effect."
else
    echo "ERROR: libvirtd failed to start. Check logs with: journalctl -u libvirtd"
    exit 1
fi
`,
		},
		{
			ID:          "arch",
			Name:        "Arch Linux",
			Description: "pacman-based install for Arch Linux and derivatives",
			Distros:     "arch,manjaro,endeavouros",
			Script: `#!/bin/bash
set -e

echo "==> Checking virtualization support..."
if [ $(egrep -c '(vmx|svm)' /proc/cpuinfo) -eq 0 ]; then
    echo "ERROR: CPU does not support virtualization (VT-x/AMD-V) or it is disabled in BIOS."
    exit 1
fi

echo "==> Installing KVM, QEMU, Libvirt and tools..."
sudo pacman -Sy --noconfirm qemu-full libvirt virt-install bridge-utils dnsmasq cdrtools wget curl

echo "==> Adding current user to libvirt group..."
sudo usermod -aG libvirt $USER 2>/dev/null || true

echo "==> Enabling and starting libvirtd..."
sudo systemctl enable --now libvirtd

echo "==> Verifying installation..."
if systemctl is-active --quiet libvirtd; then
    echo "SUCCESS: libvirtd is running."
    echo "virsh version: $(virsh --version 2>/dev/null || echo 'N/A')"
    echo "NOTE: Log out and log back in for group permissions to take effect."
else
    echo "ERROR: libvirtd failed to start. Check logs with: journalctl -u libvirtd"
    exit 1
fi
`,
		},
		{
			ID:          "opensuse",
			Name:        "openSUSE",
			Description: "zypper-based install for openSUSE Leap/Tumbleweed",
			Distros:     "opensuse,opensuse-leap,opensuse-tumbleweed,sles",
			Script: `#!/bin/bash
set -e

echo "==> Checking virtualization support..."
if [ $(egrep -c '(vmx|svm)' /proc/cpuinfo) -eq 0 ]; then
    echo "ERROR: CPU does not support virtualization (VT-x/AMD-V) or it is disabled in BIOS."
    exit 1
fi

echo "==> Installing KVM, QEMU, Libvirt and tools..."
sudo zypper install -y qemu-kvm libvirt libvirt-client virt-install bridge-utils genisoimage wget curl

echo "==> Adding current user to libvirt group..."
sudo usermod -aG libvirt $USER 2>/dev/null || true

echo "==> Enabling and starting libvirtd..."
sudo systemctl enable --now libvirtd

echo "==> Verifying installation..."
if systemctl is-active --quiet libvirtd; then
    echo "SUCCESS: libvirtd is running."
    echo "virsh version: $(virsh --version 2>/dev/null || echo 'N/A')"
    echo "NOTE: Log out and log back in for group permissions to take effect."
else
    echo "ERROR: libvirtd failed to start. Check logs with: journalctl -u libvirtd"
    exit 1
fi
`,
		},
	}
}

// HostRunScript 在远程主机上执行脚本（同步，返回输出）
func (a *App) HostRunScript(hostID, script string) (string, error) {
	client, err := a.sshPool.Get(hostID)
	if err != nil {
		return "", fmt.Errorf("host not connected: %w", err)
	}
	// 通过 heredoc 传递脚本执行
	cmd := fmt.Sprintf("bash << 'VMCAT_SCRIPT_EOF'\n%s\nVMCAT_SCRIPT_EOF", script)
	output, err := client.Execute(cmd)
	if err != nil {
		return output, fmt.Errorf("script failed: %w\nOutput: %s", err, output)
	}
	return output, nil
}

// === 镜像源管理 ===

// ImageSourceList 获取预设镜像源列表
func (a *App) ImageSourceList() ([]store.ImageSource, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.ImageSourceList()
}

// ImageSourceAdd 添加预设镜像源
func (a *App) ImageSourceAdd(src store.ImageSource) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.ImageSourceAdd(&src)
}

// ImageSourceUpdate 更新预设镜像源
func (a *App) ImageSourceUpdate(src store.ImageSource) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.ImageSourceUpdate(&src)
}

// ImageSourceDelete 删除预设镜像源
func (a *App) ImageSourceDelete(id string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.ImageSourceDelete(id)
}

// === 镜像导入 ===

// ImageImport 从 URL 下载镜像到宿主机（后台异步，通过 Events 推送进度）
func (a *App) ImageImport(hostID, url, destPath, name, osVariant string) (string, error) {
	client, err := a.sshPool.Get(hostID)
	if err != nil {
		return "", fmt.Errorf("host not connected: %w", err)
	}

	taskID := fmt.Sprintf("import-%d", time.Now().UnixNano())

	a.importMu.Lock()
	if a.importTasks == nil {
		a.importTasks = make(map[string]*importTask)
	}
	task := &importTask{ID: taskID, HostID: hostID, Status: "downloading"}
	a.importTasks[taskID] = task
	a.importMu.Unlock()

	go func() {
		defer func() {
			time.Sleep(30 * time.Second)
			a.importMu.Lock()
			delete(a.importTasks, taskID)
			a.importMu.Unlock()
		}()

		// 获取文件总大小
		sizeOut, _ := client.Execute(fmt.Sprintf(`curl -sIL '%s' | grep -i content-length | tail -1 | awk '{print $2}' | tr -d '\r\n'`, url))
		totalSize, _ := strconv.ParseInt(strings.TrimSpace(sizeOut), 10, 64)
		task.TotalSize = totalSize

		// 确保目标目录存在
		dir := destPath[:strings.LastIndex(destPath, "/")]
		client.Execute(fmt.Sprintf("mkdir -p '%s'", dir))

		// 后台下载并获取 PID
		pidOut, err := client.Execute(fmt.Sprintf(`nohup wget -q -O '%s' '%s' >/dev/null 2>&1 & echo $!`, destPath, url))
		if err != nil {
			task.Status = "error"
			task.Error = "启动下载失败: " + err.Error()
			wailsRuntime.EventsEmit(a.ctx, "image:import:error", map[string]interface{}{"taskId": taskID, "error": task.Error})
			return
		}
		pid := strings.TrimSpace(pidOut)

		// 轮询进度
		for {
			time.Sleep(2 * time.Second)

			// 检查进程是否存活
			_, aliveErr := client.Execute(fmt.Sprintf("kill -0 %s 2>/dev/null", pid))
			alive := aliveErr == nil

			// 获取当前文件大小
			curOut, _ := client.Execute(fmt.Sprintf("stat -c %%s '%s' 2>/dev/null || echo 0", destPath))
			curSize, _ := strconv.ParseInt(strings.TrimSpace(curOut), 10, 64)
			task.Current = curSize

			pct := 0
			if totalSize > 0 {
				pct = int(curSize * 100 / totalSize)
			}
			task.Percent = pct

			wailsRuntime.EventsEmit(a.ctx, "image:import:progress", map[string]interface{}{
				"taskId":    taskID,
				"percent":   pct,
				"current":   curSize,
				"totalSize": totalSize,
				"hostId":    hostID,
			})

			if !alive {
				// 验证文件是否完整
				if totalSize > 0 && curSize < totalSize*95/100 {
					task.Status = "error"
					task.Error = "下载中断或不完整"
					wailsRuntime.EventsEmit(a.ctx, "image:import:error", map[string]interface{}{"taskId": taskID, "error": task.Error})
					return
				}
				break
			}
		}

		// 下载完成，注册为 Image
		task.Status = "done"
		task.Percent = 100
		if a.store != nil {
			img := &store.Image{
				HostID:    hostID,
				Name:      name,
				BasePath:  destPath,
				OSVariant: osVariant,
			}
			a.store.ImageAdd(img)
		}
		a.audit(hostID, "", "image.import", fmt.Sprintf("url=%s dest=%s", url, destPath))
		wailsRuntime.EventsEmit(a.ctx, "image:import:done", map[string]interface{}{
			"taskId":   taskID,
			"hostId":   hostID,
			"destPath": destPath,
			"name":     name,
		})
	}()

	return taskID, nil
}

// ImageUpload 从本地文件上传镜像到宿主机（后台异步）
func (a *App) ImageUpload(hostID, localPath, destPath, name, osVariant string) (string, error) {
	client, err := a.sshPool.Get(hostID)
	if err != nil {
		return "", fmt.Errorf("host not connected: %w", err)
	}

	fi, err := os.Stat(localPath)
	if err != nil {
		return "", fmt.Errorf("本地文件不存在: %w", err)
	}
	totalSize := fi.Size()

	taskID := fmt.Sprintf("upload-%d", time.Now().UnixNano())

	a.importMu.Lock()
	if a.importTasks == nil {
		a.importTasks = make(map[string]*importTask)
	}
	task := &importTask{ID: taskID, HostID: hostID, Status: "uploading", TotalSize: totalSize}
	a.importTasks[taskID] = task
	a.importMu.Unlock()

	go func() {
		defer func() {
			time.Sleep(30 * time.Second)
			a.importMu.Lock()
			delete(a.importTasks, taskID)
			a.importMu.Unlock()
		}()

		f, err := os.Open(localPath)
		if err != nil {
			task.Status = "error"
			task.Error = "打开本地文件失败: " + err.Error()
			wailsRuntime.EventsEmit(a.ctx, "image:import:error", map[string]interface{}{"taskId": taskID, "error": task.Error})
			return
		}
		defer f.Close()

		lastEmit := time.Now()
		err = client.WriteFile(destPath, f, totalSize, func(written int64) {
			task.Current = written
			pct := int(written * 100 / totalSize)
			task.Percent = pct
			// 限制事件频率，每 500ms 推一次
			if time.Since(lastEmit) > 500*time.Millisecond {
				lastEmit = time.Now()
				wailsRuntime.EventsEmit(a.ctx, "image:import:progress", map[string]interface{}{
					"taskId":    taskID,
					"percent":   pct,
					"current":   written,
					"totalSize": totalSize,
					"hostId":    hostID,
				})
			}
		})

		if err != nil {
			task.Status = "error"
			task.Error = "上传失败: " + err.Error()
			wailsRuntime.EventsEmit(a.ctx, "image:import:error", map[string]interface{}{"taskId": taskID, "error": task.Error})
			return
		}

		// 上传完成，注册为 Image
		task.Status = "done"
		task.Percent = 100
		if a.store != nil {
			img := &store.Image{
				HostID:    hostID,
				Name:      name,
				BasePath:  destPath,
				OSVariant: osVariant,
			}
			a.store.ImageAdd(img)
		}
		a.audit(hostID, "", "image.upload", fmt.Sprintf("local=%s dest=%s", localPath, destPath))
		wailsRuntime.EventsEmit(a.ctx, "image:import:done", map[string]interface{}{
			"taskId":   taskID,
			"hostId":   hostID,
			"destPath": destPath,
			"name":     name,
		})
	}()

	return taskID, nil
}

// ImageImportStatus 获取所有活跃的导入任务状态
func (a *App) ImageImportStatus() []importTask {
	a.importMu.Lock()
	defer a.importMu.Unlock()

	var tasks []importTask
	for _, t := range a.importTasks {
		tasks = append(tasks, *t)
	}
	return tasks
}
