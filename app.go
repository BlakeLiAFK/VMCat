package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"vmcat/internal/monitor"
	internalssh "vmcat/internal/ssh"
	"vmcat/internal/store"
	"vmcat/internal/terminal"
	"vmcat/internal/vm"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed wails.json
var wailsJSON []byte

// App Wails 绑定层
type App struct {
	ctx       context.Context
	store     *store.Store
	sshPool   *internalssh.Pool
	vmManager *vm.Manager
	monitor   *monitor.Collector
	termSrv   *terminal.Server
	forceQuit bool // 真正退出标志，由托盘"退出"菜单设置
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
func (a *App) VMCreateFromTemplate(hostID, vmName, flavorID, imageID, netType, netName string) error {
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
