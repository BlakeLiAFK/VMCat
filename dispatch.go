package main

import (
	"encoding/json"
	"fmt"

	"vmcat/internal/store"
	"vmcat/internal/vm"
)

// dispatch 将 API action 路由到对应的 App 方法
func (a *App) dispatch(action string, data json.RawMessage) (interface{}, error) {
	switch action {

	// === 宿主机管理 ===

	case "host.list":
		return a.HostList()

	case "host.add":
		var p store.Host
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.HostAdd(p)

	case "host.update":
		var p store.Host
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.HostUpdate(p)

	case "host.delete":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.HostDelete(p.ID)

	case "host.connect":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.HostConnect(p.ID)

	case "host.disconnect":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		a.HostDisconnect(p.ID)
		return nil, nil

	case "host.test":
		var p store.Host
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostTest(p)

	case "host.resetHostKey":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.HostResetHostKey(p.ID)

	case "host.getFingerprint":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostGetFingerprint(p.ID)

	case "host.isConnected":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostIsConnected(p.ID), nil

	case "host.resourceStats":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostResourceStats(p.HostID)

	case "host.exportJSON":
		return a.HostExportJSON()

	case "host.importJSON":
		var p struct {
			JSON string `json:"json"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostImportJSON(p.JSON)

	case "host.checkTools":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostCheckTools(p.ID)

	case "host.detectDistro":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostDetectDistro(p.ID)

	case "host.runScript":
		var p struct {
			HostID string `json:"hostId"`
			Script string `json:"script"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostRunScript(p.HostID, p.Script)

	case "host.imageScan":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostImageScan(p.HostID)

	case "host.imageDelete":
		var p struct {
			HostID string `json:"hostId"`
			Path   string `json:"path"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.HostImageDelete(p.HostID, p.Path)

	case "host.statsHistory":
		var p struct {
			HostID string `json:"hostId"`
			Hours  int    `json:"hours"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.HostStatsHistory(p.HostID, p.Hours)

	// === VM 管理 ===

	case "vm.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VMList(p.HostID)

	case "vm.get":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VMGet(p.HostID, p.VMName)

	case "vm.start":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMStart(p.HostID, p.VMName)

	case "vm.shutdown":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMShutdown(p.HostID, p.VMName)

	case "vm.destroy":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMDestroy(p.HostID, p.VMName)

	case "vm.reboot":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMReboot(p.HostID, p.VMName)

	case "vm.suspend":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMSuspend(p.HostID, p.VMName)

	case "vm.resume":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMResume(p.HostID, p.VMName)

	case "vm.delete":
		var p struct {
			HostID        string `json:"hostId"`
			VMName        string `json:"vmName"`
			RemoveStorage bool   `json:"removeStorage"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMDelete(p.HostID, p.VMName, p.RemoveStorage)

	case "vm.rename":
		var p struct {
			HostID  string `json:"hostId"`
			OldName string `json:"oldName"`
			NewName string `json:"newName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMRename(p.HostID, p.OldName, p.NewName)

	case "vm.setVCPUs":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			Count  int    `json:"count"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMSetVCPUs(p.HostID, p.VMName, p.Count)

	case "vm.setMemory":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			SizeMB int    `json:"sizeMB"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMSetMemory(p.HostID, p.VMName, p.SizeMB)

	case "vm.setAutostart":
		var p struct {
			HostID  string `json:"hostId"`
			VMName  string `json:"vmName"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMSetAutostart(p.HostID, p.VMName, p.Enabled)

	case "vm.clone":
		var p struct {
			HostID  string `json:"hostId"`
			SrcName string `json:"srcName"`
			NewName string `json:"newName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMClone(p.HostID, p.SrcName, p.NewName)

	case "vm.getXML":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VMGetXML(p.HostID, p.VMName)

	case "vm.defineXML":
		var p struct {
			HostID     string `json:"hostId"`
			XMLContent string `json:"xmlContent"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMDefineXML(p.HostID, p.XMLContent)

	case "vm.create":
		var p struct {
			HostID string          `json:"hostId"`
			Params vm.VMCreateParams `json:"params"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMCreate(p.HostID, p.Params)

	case "vm.stats":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VMStats(p.HostID, p.VMName)

	case "vm.createFromTemplate":
		var p struct {
			HostID       string `json:"hostId"`
			VMName       string `json:"vmName"`
			FlavorID     string `json:"flavorId"`
			ImageID      string `json:"imageId"`
			NetType      string `json:"netType"`
			NetName      string `json:"netName"`
			RootPassword string `json:"rootPassword"`
			SSHPubKey    string `json:"sshPubKey"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMCreateFromTemplate(p.HostID, p.VMName, p.FlavorID, p.ImageID, p.NetType, p.NetName, p.RootPassword, p.SSHPubKey)

	case "vm.migrate":
		var p struct {
			SrcHostID string `json:"srcHostId"`
			VMName    string `json:"vmName"`
			DstHostID string `json:"dstHostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMMigrate(p.SrcHostID, p.VMName, p.DstHostID)

	case "vm.migrateOffline":
		var p struct {
			SrcHostID string `json:"srcHostId"`
			VMName    string `json:"vmName"`
			DstHostID string `json:"dstHostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMMigrateOffline(p.SrcHostID, p.VMName, p.DstHostID)

	case "vm.noteGet":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VMNoteGet(p.HostID, p.VMName)

	case "vm.noteSet":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			Note   string `json:"note"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMNoteSet(p.HostID, p.VMName, p.Note)

	case "vm.statsHistory":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			Hours  int    `json:"hours"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VMStatsHistory(p.HostID, p.VMName, p.Hours)

	// === 硬件管理 ===

	case "vm.attachDisk":
		var p struct {
			HostID string             `json:"hostId"`
			VMName string             `json:"vmName"`
			Params vm.DiskAttachParams `json:"params"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMAttachDisk(p.HostID, p.VMName, p.Params)

	case "vm.detachDisk":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			Target string `json:"target"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMDetachDisk(p.HostID, p.VMName, p.Target)

	case "vm.resizeDisk":
		var p struct {
			HostID    string `json:"hostId"`
			DiskPath  string `json:"diskPath"`
			NewSizeGB int    `json:"newSizeGB"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMResizeDisk(p.HostID, p.DiskPath, p.NewSizeGB)

	case "vm.attachInterface":
		var p struct {
			HostID string              `json:"hostId"`
			VMName string              `json:"vmName"`
			Params vm.NICAttachParams  `json:"params"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMAttachInterface(p.HostID, p.VMName, p.Params)

	case "vm.detachInterface":
		var p struct {
			HostID  string `json:"hostId"`
			VMName  string `json:"vmName"`
			MacAddr string `json:"macAddr"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMDetachInterface(p.HostID, p.VMName, p.MacAddr)

	case "vm.changeMedia":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			Target string `json:"target"`
			Source string `json:"source"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMChangeMedia(p.HostID, p.VMName, p.Target, p.Source)

	case "vm.ejectMedia":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
			Target string `json:"target"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMEjectMedia(p.HostID, p.VMName, p.Target)

	case "vm.setGraphics":
		var p struct {
			HostID  string `json:"hostId"`
			VMName  string `json:"vmName"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMSetGraphics(p.HostID, p.VMName, p.Enabled)

	case "vm.generateCloudInit":
		var p struct {
			HostID     string             `json:"hostId"`
			OutputPath string             `json:"outputPath"`
			Config     vm.CloudInitConfig `json:"config"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.VMGenerateCloudInit(p.HostID, p.OutputPath, p.Config)

	// === 快照管理 ===

	case "snapshot.list":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.SnapshotList(p.HostID, p.VMName)

	case "snapshot.create":
		var p struct {
			HostID   string `json:"hostId"`
			VMName   string `json:"vmName"`
			SnapName string `json:"snapName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.SnapshotCreate(p.HostID, p.VMName, p.SnapName)

	case "snapshot.delete":
		var p struct {
			HostID   string `json:"hostId"`
			VMName   string `json:"vmName"`
			SnapName string `json:"snapName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.SnapshotDelete(p.HostID, p.VMName, p.SnapName)

	case "snapshot.revert":
		var p struct {
			HostID   string `json:"hostId"`
			VMName   string `json:"vmName"`
			SnapName string `json:"snapName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.SnapshotRevert(p.HostID, p.VMName, p.SnapName)

	// === 存储管理 ===

	case "pool.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.PoolList(p.HostID)

	case "pool.start":
		var p struct {
			HostID   string `json:"hostId"`
			PoolName string `json:"poolName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.PoolStart(p.HostID, p.PoolName)

	case "pool.stop":
		var p struct {
			HostID   string `json:"hostId"`
			PoolName string `json:"poolName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.PoolStop(p.HostID, p.PoolName)

	case "pool.autostart":
		var p struct {
			HostID   string `json:"hostId"`
			PoolName string `json:"poolName"`
			Enabled  bool   `json:"enabled"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.PoolAutostart(p.HostID, p.PoolName, p.Enabled)

	case "vol.list":
		var p struct {
			HostID   string `json:"hostId"`
			PoolName string `json:"poolName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.VolList(p.HostID, p.PoolName)

	case "vol.create":
		var p struct {
			HostID   string `json:"hostId"`
			PoolName string `json:"poolName"`
			VolName  string `json:"volName"`
			SizeGB   int    `json:"sizeGB"`
			Format   string `json:"format"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.CreateVolume(p.HostID, p.PoolName, p.VolName, p.SizeGB, p.Format)

	case "vol.delete":
		var p struct {
			HostID   string `json:"hostId"`
			PoolName string `json:"poolName"`
			VolName  string `json:"volName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.DeleteVolume(p.HostID, p.PoolName, p.VolName)

	// === 网络管理 ===

	case "network.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.NetworkList(p.HostID)

	case "network.start":
		var p struct {
			HostID  string `json:"hostId"`
			NetName string `json:"netName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.NetworkStart(p.HostID, p.NetName)

	case "network.stop":
		var p struct {
			HostID  string `json:"hostId"`
			NetName string `json:"netName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.NetworkStop(p.HostID, p.NetName)

	case "network.autostart":
		var p struct {
			HostID  string `json:"hostId"`
			NetName string `json:"netName"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.NetworkAutostart(p.HostID, p.NetName, p.Enabled)

	case "bridge.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.BridgeList(p.HostID)

	// === NAT 端口转发 ===

	case "nat.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.NATRuleList(p.HostID)

	case "nat.add":
		var p struct {
			HostID   string `json:"hostId"`
			Proto    string `json:"proto"`
			HostPort string `json:"hostPort"`
			VMIP     string `json:"vmIP"`
			VMPort   string `json:"vmPort"`
			Comment  string `json:"comment"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.NATRuleAdd(p.HostID, p.Proto, p.HostPort, p.VMIP, p.VMPort, p.Comment)

	case "nat.delete":
		var p struct {
			HostID   string `json:"hostId"`
			Proto    string `json:"proto"`
			HostPort string `json:"hostPort"`
			VMIP     string `json:"vmIP"`
			VMPort   string `json:"vmPort"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.NATRuleDelete(p.HostID, p.Proto, p.HostPort, p.VMIP, p.VMPort)

	// === ISO / OS Variant ===

	case "iso.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.ISOList(p.HostID)

	case "osvariant.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.OSVariantList(p.HostID)

	// === 模板管理 ===

	case "flavor.list":
		return a.FlavorList()

	case "flavor.add":
		var p store.Flavor
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.FlavorAdd(p)

	case "flavor.update":
		var p store.Flavor
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.FlavorUpdate(p)

	case "flavor.delete":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.FlavorDelete(p.ID)

	case "image.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.ImageList(p.HostID)

	case "image.add":
		var p struct {
			HostID string      `json:"hostId"`
			Image  store.Image `json:"image"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.ImageAdd(p.HostID, p.Image)

	case "image.update":
		var p store.Image
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.ImageUpdate(p)

	case "image.delete":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.ImageDelete(p.ID)

	case "image.import":
		var p struct {
			HostID    string `json:"hostId"`
			URL       string `json:"url"`
			DestPath  string `json:"destPath"`
			Name      string `json:"name"`
			OSVariant string `json:"osVariant"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.ImageImport(p.HostID, p.URL, p.DestPath, p.Name, p.OSVariant)

	case "image.upload":
		// 远程模式暂不支持本地文件上传
		return nil, fmt.Errorf("image.upload is not supported in remote mode")

	case "image.importStatus":
		return a.ImageImportStatus(), nil

	// === 镜像源管理 ===

	case "imageSource.list":
		return a.ImageSourceList()

	case "imageSource.add":
		var p store.ImageSource
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.ImageSourceAdd(p)

	case "imageSource.update":
		var p store.ImageSource
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.ImageSourceUpdate(p)

	case "imageSource.delete":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.ImageSourceDelete(p.ID)

	// === Instance 管理 ===

	case "instance.list":
		var p struct {
			HostID string `json:"hostId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.InstanceList(p.HostID)

	case "instance.byVMName":
		var p struct {
			HostID string `json:"hostId"`
			VMName string `json:"vmName"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.InstanceByVMName(p.HostID, p.VMName)

	case "instance.isoList":
		var p struct {
			HostID     string `json:"hostId"`
			InstanceID int    `json:"instanceId"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.InstanceISOList(p.HostID, p.InstanceID)

	// === 设置 ===

	case "setting.get":
		var p struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.SettingGet(p.Key)

	case "setting.set":
		var p struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return nil, a.SettingSet(p.Key, p.Value)

	// === 审计日志 ===

	case "audit.list":
		var p struct {
			HostID string `json:"hostId"`
			Limit  int    `json:"limit"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.AuditList(p.HostID, p.Limit)

	case "audit.listAll":
		var p struct {
			Limit int `json:"limit"`
		}
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		return a.AuditListAll(p.Limit)

	// === 工具 ===

	case "app.version":
		return a.AppVersion(), nil

	case "terminal.port":
		return a.TerminalPort(), nil

	case "libvirt.setupScripts":
		return a.LibvirtSetupScriptList(), nil

	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
