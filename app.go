package main

import (
	"context"
	"fmt"
	"log"

	"vmcat/internal/monitor"
	internalssh "vmcat/internal/ssh"
	"vmcat/internal/store"
	"vmcat/internal/terminal"
	"vmcat/internal/vm"
)

// App Wails 绑定层
type App struct {
	ctx       context.Context
	store     *store.Store
	sshPool   *internalssh.Pool
	vmManager *vm.Manager
	monitor   *monitor.Collector
	termSrv   *terminal.Server
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

	// 启动终端 WebSocket 服务
	if err := a.termSrv.Start(); err != nil {
		log.Printf("start terminal server: %v", err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.termSrv.Close()
	a.sshPool.CloseAll()
	if a.store != nil {
		a.store.Close()
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
		Host:     h.Host,
		Port:     h.Port,
		User:     h.User,
		AuthType: h.AuthType,
		KeyPath:  h.KeyPath,
		Password: h.Password,
	}

	_, err = a.sshPool.Connect(id, cfg)
	return err
}

// HostDisconnect 断开宿主机
func (a *App) HostDisconnect(id string) {
	a.sshPool.Disconnect(id)
}

// HostTest 测试 SSH 连接
func (a *App) HostTest(h store.Host) error {
	cfg := &internalssh.Config{
		Host:     h.Host,
		Port:     h.Port,
		User:     h.User,
		AuthType: h.AuthType,
		KeyPath:  h.KeyPath,
		Password: h.Password,
	}

	client := internalssh.NewClient(cfg)
	if err := client.Connect(); err != nil {
		return err
	}

	output, err := client.Execute("hostname")
	if err != nil {
		client.Close()
		return fmt.Errorf("execute test command: %w", err)
	}

	log.Printf("host test ok: %s", output)
	client.Close()
	return nil
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
	return a.vmManager.Start(hostID, vmName)
}

// VMShutdown 关闭虚拟机
func (a *App) VMShutdown(hostID, vmName string) error {
	return a.vmManager.Shutdown(hostID, vmName)
}

// VMDestroy 强制关闭虚拟机
func (a *App) VMDestroy(hostID, vmName string) error {
	return a.vmManager.Destroy(hostID, vmName)
}

// VMReboot 重启虚拟机
func (a *App) VMReboot(hostID, vmName string) error {
	return a.vmManager.Reboot(hostID, vmName)
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
	return a.vmManager.Delete(hostID, vmName, removeStorage)
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
	return a.vmManager.Clone(hostID, srcName, newName)
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
	return a.vmManager.Create(hostID, params)
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

// === 网络管理 ===

// NetworkList 获取网络列表
func (a *App) NetworkList(hostID string) ([]vm.Network, error) {
	return a.vmManager.NetworkList(hostID)
}

// BridgeList 获取网桥列表
func (a *App) BridgeList(hostID string) ([]string, error) {
	return a.vmManager.BridgeList(hostID)
}

// === ISO 管理 ===

// ISOList 列出 ISO 镜像
func (a *App) ISOList(hostID string) ([]vm.ISOFile, error) {
	return a.vmManager.ISOList(hostID, nil)
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
