// 前端后端抽象层 - 本地/远程模式透明切换
// 所有组件通过此模块调用后端,而非直接调用 Wails 绑定

import * as WailsAPI from '../../wailsjs/go/main/App'
import { RemoteClient } from './remote'
import type { RemoteConfig } from './types'

// 导入类型定义
import type { store, main, monitor, vm } from '../../wailsjs/go/models'

// --- 状态管理 ---
let remoteClient: RemoteClient | null = null
let _mode: 'local' | 'remote' = 'local'

export function isRemoteMode(): boolean {
  return _mode === 'remote'
}

export function getMode(): 'local' | 'remote' {
  return _mode
}

export function switchToRemote(config: RemoteConfig): void {
  remoteClient = new RemoteClient(config)
  _mode = 'remote'
}

export function switchToLocal(): void {
  remoteClient = null
  _mode = 'local'
}

export function getRemoteClient(): RemoteClient | null {
  return remoteClient
}

// --- WebSocket URL 生成 ---

export async function getTerminalWSURL(params: Record<string, string>): Promise<string> {
  if (remoteClient) {
    return remoteClient.wsURL('/ws/terminal', params)
  }
  const port = await WailsAPI.TerminalPort()
  const qs = new URLSearchParams(params)
  return `ws://127.0.0.1:${port}/ws/terminal?${qs.toString()}`
}

export async function getVNCWSURL(params: Record<string, string>): Promise<string> {
  if (remoteClient) {
    return remoteClient.wsURL('/ws/vnc', params)
  }
  const port = await WailsAPI.TerminalPort()
  const qs = new URLSearchParams(params)
  return `ws://127.0.0.1:${port}/ws/vnc?${qs.toString()}`
}

// --- 应用 API ---

export async function AppVersion(): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('app.version')
  }
  return WailsAPI.AppVersion()
}

// --- 审计日志 API ---

export async function AuditList(hostId: string, limit: number): Promise<store.AuditRecord[]> {
  if (remoteClient) {
    return remoteClient.call('audit.list', { hostId, limit })
  }
  return WailsAPI.AuditList(hostId, limit)
}

export async function AuditListAll(limit: number): Promise<store.AuditRecord[]> {
  if (remoteClient) {
    return remoteClient.call('audit.listAll', { limit })
  }
  return WailsAPI.AuditListAll(limit)
}

// --- 网桥 API ---

export async function BridgeList(hostId: string): Promise<string[]> {
  if (remoteClient) {
    return remoteClient.call('bridge.list', { hostId })
  }
  return WailsAPI.BridgeList(hostId)
}

// --- 存储卷 API ---

export async function CreateVolume(
  hostId: string,
  pool: string,
  name: string,
  sizeGB: number,
  format: string
): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('vol.create', {
      hostId,
      poolName: pool,
      volName: name,
      sizeGB,
      format,
    })
  }
  return WailsAPI.CreateVolume(hostId, pool, name, sizeGB, format)
}

export async function DeleteVolume(hostId: string, pool: string, name: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vol.delete', {
      hostId,
      poolName: pool,
      volName: name,
    })
  }
  return WailsAPI.DeleteVolume(hostId, pool, name)
}

export async function VolList(hostId: string, poolName: string): Promise<vm.Volume[]> {
  if (remoteClient) {
    return remoteClient.call('vol.list', { hostId, poolName })
  }
  return WailsAPI.VolList(hostId, poolName)
}

// --- 配置规格 API ---

export async function FlavorAdd(flavor: store.Flavor): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('flavor.add', flavor)
  }
  return WailsAPI.FlavorAdd(flavor)
}

export async function FlavorDelete(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('flavor.delete', { id })
  }
  return WailsAPI.FlavorDelete(id)
}

export async function FlavorList(): Promise<store.Flavor[]> {
  if (remoteClient) {
    return remoteClient.call('flavor.list')
  }
  return WailsAPI.FlavorList()
}

export async function FlavorUpdate(flavor: store.Flavor): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('flavor.update', flavor)
  }
  return WailsAPI.FlavorUpdate(flavor)
}

// --- 宿主机 API ---

export async function HostAdd(host: store.Host): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.add', host)
  }
  return WailsAPI.HostAdd(host)
}

export async function HostCheckTools(id: string): Promise<Record<string, string>> {
  if (remoteClient) {
    return remoteClient.call('host.checkTools', { id })
  }
  return WailsAPI.HostCheckTools(id)
}

export async function HostConnect(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.connect', { id })
  }
  return WailsAPI.HostConnect(id)
}

export async function HostDelete(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.delete', { id })
  }
  return WailsAPI.HostDelete(id)
}

export async function HostDetectDistro(id: string): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('host.detectDistro', { id })
  }
  return WailsAPI.HostDetectDistro(id)
}

export async function HostDisconnect(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.disconnect', { id })
  }
  return WailsAPI.HostDisconnect(id)
}

export async function HostExportJSON(): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('host.exportJSON')
  }
  return WailsAPI.HostExportJSON()
}

export async function HostGetFingerprint(id: string): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('host.getFingerprint', { id })
  }
  return WailsAPI.HostGetFingerprint(id)
}

export async function HostImageDelete(hostId: string, path: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.imageDelete', { hostId, path })
  }
  return WailsAPI.HostImageDelete(hostId, path)
}

export async function HostImageScan(hostId: string): Promise<main.HostImageFile[]> {
  if (remoteClient) {
    return remoteClient.call('host.imageScan', { hostId })
  }
  return WailsAPI.HostImageScan(hostId)
}

export async function HostImportJSON(json: string): Promise<number> {
  if (remoteClient) {
    return remoteClient.call('host.importJSON', { json })
  }
  return WailsAPI.HostImportJSON(json)
}

export async function HostIsConnected(id: string): Promise<boolean> {
  if (remoteClient) {
    return remoteClient.call('host.isConnected', { id })
  }
  return WailsAPI.HostIsConnected(id)
}

export async function HostList(): Promise<store.Host[]> {
  if (remoteClient) {
    return remoteClient.call('host.list')
  }
  return WailsAPI.HostList()
}

export async function HostResetHostKey(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.resetHostKey', { id })
  }
  return WailsAPI.HostResetHostKey(id)
}

export async function HostResourceStats(hostId: string): Promise<monitor.HostStats> {
  if (remoteClient) {
    return remoteClient.call('host.resourceStats', { hostId })
  }
  return WailsAPI.HostResourceStats(hostId)
}

export async function HostRunScript(hostId: string, script: string): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('host.runScript', { hostId, script })
  }
  return WailsAPI.HostRunScript(hostId, script)
}

export async function HostStatsHistory(
  hostId: string,
  hours: number
): Promise<store.HostStatsRecord[]> {
  if (remoteClient) {
    return remoteClient.call('host.statsHistory', { hostId, hours })
  }
  return WailsAPI.HostStatsHistory(hostId, hours)
}

export async function HostTest(host: store.Host): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('host.test', host)
  }
  return WailsAPI.HostTest(host)
}

export async function HostUpdate(host: store.Host): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('host.update', host)
  }
  return WailsAPI.HostUpdate(host)
}

// --- ISO 文件 API ---

export async function ISOList(hostId: string): Promise<vm.ISOFile[]> {
  if (remoteClient) {
    return remoteClient.call('iso.list', { hostId })
  }
  return WailsAPI.ISOList(hostId)
}

// --- 镜像 API ---

export async function ImageAdd(hostId: string, image: store.Image): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('image.add', { hostId, image })
  }
  return WailsAPI.ImageAdd(hostId, image)
}

export async function ImageDelete(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('image.delete', { id })
  }
  return WailsAPI.ImageDelete(id)
}

export async function ImageImport(
  hostId: string,
  url: string,
  destPath: string,
  name: string,
  osVariant: string
): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('image.import', {
      hostId,
      url,
      destPath,
      name,
      osVariant,
    })
  }
  return WailsAPI.ImageImport(hostId, url, destPath, name, osVariant)
}

export async function ImageImportStatus(): Promise<main.importTask[]> {
  if (remoteClient) {
    return remoteClient.call('image.importStatus')
  }
  return WailsAPI.ImageImportStatus()
}

export async function ImageList(hostId: string): Promise<store.Image[]> {
  if (remoteClient) {
    return remoteClient.call('image.list', { hostId })
  }
  return WailsAPI.ImageList(hostId)
}

export async function ImageUpdate(image: store.Image): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('image.update', image)
  }
  return WailsAPI.ImageUpdate(image)
}

export async function ImageUpload(
  hostId: string,
  localPath: string,
  destPath: string,
  name: string,
  osVariant: string
): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('image.upload', {
      hostId,
      localPath,
      destPath,
      name,
      osVariant,
    })
  }
  return WailsAPI.ImageUpload(hostId, localPath, destPath, name, osVariant)
}

// --- 镜像源 API ---

export async function ImageSourceAdd(source: store.ImageSource): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('imageSource.add', source)
  }
  return WailsAPI.ImageSourceAdd(source)
}

export async function ImageSourceDelete(id: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('imageSource.delete', { id })
  }
  return WailsAPI.ImageSourceDelete(id)
}

export async function ImageSourceList(): Promise<store.ImageSource[]> {
  if (remoteClient) {
    return remoteClient.call('imageSource.list')
  }
  return WailsAPI.ImageSourceList()
}

export async function ImageSourceUpdate(source: store.ImageSource): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('imageSource.update', source)
  }
  return WailsAPI.ImageSourceUpdate(source)
}

// --- 实例 API ---

export async function InstanceByVMName(hostId: string, vmName: string): Promise<store.Instance> {
  if (remoteClient) {
    return remoteClient.call('instance.byVMName', { hostId, vmName })
  }
  return WailsAPI.InstanceByVMName(hostId, vmName)
}

export async function InstanceISOList(hostId: string, instanceId: number): Promise<vm.ISOFile[]> {
  if (remoteClient) {
    return remoteClient.call('instance.isoList', { hostId, instanceId })
  }
  return WailsAPI.InstanceISOList(hostId, instanceId)
}

export async function InstanceList(hostId: string): Promise<store.Instance[]> {
  if (remoteClient) {
    return remoteClient.call('instance.list', { hostId })
  }
  return WailsAPI.InstanceList(hostId)
}

// --- Libvirt 安装脚本 API ---

export async function LibvirtSetupScriptList(): Promise<main.LibvirtSetupScript[]> {
  if (remoteClient) {
    return remoteClient.call('libvirt.setupScripts')
  }
  return WailsAPI.LibvirtSetupScriptList()
}

// --- NAT 规则 API ---

export async function NATRuleAdd(
  hostId: string,
  proto: string,
  hostPort: string,
  vmIP: string,
  vmPort: string,
  comment: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('nat.add', {
      hostId,
      proto,
      hostPort,
      vmIP,
      vmPort,
      comment,
    })
  }
  return WailsAPI.NATRuleAdd(hostId, proto, hostPort, vmIP, vmPort, comment)
}

export async function NATRuleDelete(
  hostId: string,
  proto: string,
  hostPort: string,
  vmIP: string,
  vmPort: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('nat.delete', {
      hostId,
      proto,
      hostPort,
      vmIP,
      vmPort,
    })
  }
  return WailsAPI.NATRuleDelete(hostId, proto, hostPort, vmIP, vmPort)
}

export async function NATRuleList(hostId: string): Promise<vm.NATRule[]> {
  if (remoteClient) {
    return remoteClient.call('nat.list', { hostId })
  }
  return WailsAPI.NATRuleList(hostId)
}

// --- 网络 API ---

export async function NetworkAutostart(
  hostId: string,
  name: string,
  enabled: boolean
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('network.autostart', {
      hostId,
      netName: name,
      enabled,
    })
  }
  return WailsAPI.NetworkAutostart(hostId, name, enabled)
}

export async function NetworkList(hostId: string): Promise<vm.Network[]> {
  if (remoteClient) {
    return remoteClient.call('network.list', { hostId })
  }
  return WailsAPI.NetworkList(hostId)
}

export async function NetworkStart(hostId: string, name: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('network.start', { hostId, netName: name })
  }
  return WailsAPI.NetworkStart(hostId, name)
}

export async function NetworkStop(hostId: string, name: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('network.stop', { hostId, netName: name })
  }
  return WailsAPI.NetworkStop(hostId, name)
}

// --- OS Variant API ---

export async function OSVariantList(hostId: string): Promise<string[]> {
  if (remoteClient) {
    return remoteClient.call('osvariant.list', { hostId })
  }
  return WailsAPI.OSVariantList(hostId)
}

// --- 存储池 API ---

export async function PoolAutostart(hostId: string, name: string, enabled: boolean): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('pool.autostart', {
      hostId,
      poolName: name,
      enabled,
    })
  }
  return WailsAPI.PoolAutostart(hostId, name, enabled)
}

export async function PoolList(hostId: string): Promise<vm.StoragePool[]> {
  if (remoteClient) {
    return remoteClient.call('pool.list', { hostId })
  }
  return WailsAPI.PoolList(hostId)
}

export async function PoolStart(hostId: string, name: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('pool.start', { hostId, poolName: name })
  }
  return WailsAPI.PoolStart(hostId, name)
}

export async function PoolStop(hostId: string, name: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('pool.stop', { hostId, poolName: name })
  }
  return WailsAPI.PoolStop(hostId, name)
}

// --- 设置 API ---

export async function SettingGet(key: string): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('setting.get', { key })
  }
  return WailsAPI.SettingGet(key)
}

export async function SettingSet(key: string, value: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('setting.set', { key, value })
  }
  return WailsAPI.SettingSet(key, value)
}

// --- 快照 API ---

export async function SnapshotCreate(
  hostId: string,
  vmName: string,
  snapName: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('snapshot.create', { hostId, vmName, snapName })
  }
  return WailsAPI.SnapshotCreate(hostId, vmName, snapName)
}

export async function SnapshotDelete(
  hostId: string,
  vmName: string,
  snapName: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('snapshot.delete', { hostId, vmName, snapName })
  }
  return WailsAPI.SnapshotDelete(hostId, vmName, snapName)
}

export async function SnapshotList(hostId: string, vmName: string): Promise<vm.Snapshot[]> {
  if (remoteClient) {
    return remoteClient.call('snapshot.list', { hostId, vmName })
  }
  return WailsAPI.SnapshotList(hostId, vmName)
}

export async function SnapshotRevert(
  hostId: string,
  vmName: string,
  snapName: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('snapshot.revert', { hostId, vmName, snapName })
  }
  return WailsAPI.SnapshotRevert(hostId, vmName, snapName)
}

// --- 终端 API ---

export async function TerminalPort(): Promise<number> {
  if (remoteClient) {
    return remoteClient.call('terminal.port')
  }
  return WailsAPI.TerminalPort()
}

// --- 虚拟机 API ---

export async function VMAttachDisk(
  hostId: string,
  vmName: string,
  params: vm.DiskAttachParams
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.attachDisk', { hostId, vmName, params })
  }
  return WailsAPI.VMAttachDisk(hostId, vmName, params)
}

export async function VMAttachInterface(
  hostId: string,
  vmName: string,
  params: vm.NICAttachParams
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.attachInterface', { hostId, vmName, params })
  }
  return WailsAPI.VMAttachInterface(hostId, vmName, params)
}

export async function VMChangeMedia(
  hostId: string,
  vmName: string,
  target: string,
  source: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.changeMedia', {
      hostId,
      vmName,
      target,
      source,
    })
  }
  return WailsAPI.VMChangeMedia(hostId, vmName, target, source)
}

export async function VMClone(hostId: string, srcName: string, newName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.clone', { hostId, srcName, newName })
  }
  return WailsAPI.VMClone(hostId, srcName, newName)
}

export async function VMCreate(hostId: string, params: vm.VMCreateParams): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.create', { hostId, params })
  }
  return WailsAPI.VMCreate(hostId, params)
}

export async function VMCreateFromTemplate(
  hostId: string,
  vmName: string,
  flavorId: string,
  imageId: string,
  netType: string,
  netName: string,
  rootPassword: string,
  sshPubKey: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.createFromTemplate', {
      hostId,
      vmName,
      flavorId,
      imageId,
      netType,
      netName,
      rootPassword,
      sshPubKey,
    })
  }
  return WailsAPI.VMCreateFromTemplate(
    hostId,
    vmName,
    flavorId,
    imageId,
    netType,
    netName,
    rootPassword,
    sshPubKey
  )
}

export async function VMDefineXML(hostId: string, xml: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.defineXML', { hostId, xmlContent: xml })
  }
  return WailsAPI.VMDefineXML(hostId, xml)
}

export async function VMDelete(
  hostId: string,
  vmName: string,
  removeStorage: boolean
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.delete', { hostId, vmName, removeStorage })
  }
  return WailsAPI.VMDelete(hostId, vmName, removeStorage)
}

export async function VMDestroy(hostId: string, vmName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.destroy', { hostId, vmName })
  }
  return WailsAPI.VMDestroy(hostId, vmName)
}

export async function VMDetachDisk(hostId: string, vmName: string, target: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.detachDisk', { hostId, vmName, target })
  }
  return WailsAPI.VMDetachDisk(hostId, vmName, target)
}

export async function VMDetachInterface(
  hostId: string,
  vmName: string,
  macAddr: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.detachInterface', { hostId, vmName, macAddr })
  }
  return WailsAPI.VMDetachInterface(hostId, vmName, macAddr)
}

export async function VMEjectMedia(hostId: string, vmName: string, target: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.ejectMedia', { hostId, vmName, target })
  }
  return WailsAPI.VMEjectMedia(hostId, vmName, target)
}

export async function VMGenerateCloudInit(
  hostId: string,
  outputPath: string,
  config: vm.CloudInitConfig
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.generateCloudInit', {
      hostId,
      outputPath,
      config,
    })
  }
  return WailsAPI.VMGenerateCloudInit(hostId, outputPath, config)
}

export async function VMGet(hostId: string, vmName: string): Promise<vm.VMDetail> {
  if (remoteClient) {
    return remoteClient.call('vm.get', { hostId, vmName })
  }
  return WailsAPI.VMGet(hostId, vmName)
}

export async function VMGetXML(hostId: string, vmName: string): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('vm.getXML', { hostId, vmName })
  }
  return WailsAPI.VMGetXML(hostId, vmName)
}

export async function VMList(hostId: string): Promise<vm.VM[]> {
  if (remoteClient) {
    return remoteClient.call('vm.list', { hostId })
  }
  return WailsAPI.VMList(hostId)
}

export async function VMMigrate(
  srcHostId: string,
  vmName: string,
  dstHostId: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.migrate', { srcHostId, vmName, dstHostId })
  }
  return WailsAPI.VMMigrate(srcHostId, vmName, dstHostId)
}

export async function VMMigrateOffline(
  srcHostId: string,
  vmName: string,
  dstHostId: string
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.migrateOffline', { srcHostId, vmName, dstHostId })
  }
  return WailsAPI.VMMigrateOffline(srcHostId, vmName, dstHostId)
}

export async function VMNoteGet(hostId: string, vmName: string): Promise<string> {
  if (remoteClient) {
    return remoteClient.call('vm.noteGet', { hostId, vmName })
  }
  return WailsAPI.VMNoteGet(hostId, vmName)
}

export async function VMNoteSet(hostId: string, vmName: string, note: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.noteSet', { hostId, vmName, note })
  }
  return WailsAPI.VMNoteSet(hostId, vmName, note)
}

export async function VMReboot(hostId: string, vmName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.reboot', { hostId, vmName })
  }
  return WailsAPI.VMReboot(hostId, vmName)
}

export async function VMRename(hostId: string, oldName: string, newName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.rename', { hostId, oldName, newName })
  }
  return WailsAPI.VMRename(hostId, oldName, newName)
}

export async function VMResizeDisk(
  hostId: string,
  diskPath: string,
  newSizeGB: number
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.resizeDisk', { hostId, diskPath, newSizeGB })
  }
  return WailsAPI.VMResizeDisk(hostId, diskPath, newSizeGB)
}

export async function VMResume(hostId: string, vmName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.resume', { hostId, vmName })
  }
  return WailsAPI.VMResume(hostId, vmName)
}

export async function VMSetAutostart(
  hostId: string,
  vmName: string,
  enabled: boolean
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.setAutostart', { hostId, vmName, enabled })
  }
  return WailsAPI.VMSetAutostart(hostId, vmName, enabled)
}

export async function VMSetGraphics(
  hostId: string,
  vmName: string,
  enabled: boolean
): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.setGraphics', { hostId, vmName, enabled })
  }
  return WailsAPI.VMSetGraphics(hostId, vmName, enabled)
}

export async function VMSetMemory(hostId: string, vmName: string, sizeMB: number): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.setMemory', { hostId, vmName, sizeMB })
  }
  return WailsAPI.VMSetMemory(hostId, vmName, sizeMB)
}

export async function VMSetVCPUs(hostId: string, vmName: string, count: number): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.setVCPUs', { hostId, vmName, count })
  }
  return WailsAPI.VMSetVCPUs(hostId, vmName, count)
}

export async function VMShutdown(hostId: string, vmName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.shutdown', { hostId, vmName })
  }
  return WailsAPI.VMShutdown(hostId, vmName)
}

export async function VMStart(hostId: string, vmName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.start', { hostId, vmName })
  }
  return WailsAPI.VMStart(hostId, vmName)
}

export async function VMStats(hostId: string, vmName: string): Promise<vm.VMResourceStats> {
  if (remoteClient) {
    return remoteClient.call('vm.stats', { hostId, vmName })
  }
  return WailsAPI.VMStats(hostId, vmName)
}

export async function VMStatsHistory(
  hostId: string,
  vmName: string,
  hours: number
): Promise<store.VMStatsRecord[]> {
  if (remoteClient) {
    return remoteClient.call('vm.statsHistory', { hostId, vmName, hours })
  }
  return WailsAPI.VMStatsHistory(hostId, vmName, hours)
}

export async function VMSuspend(hostId: string, vmName: string): Promise<void> {
  if (remoteClient) {
    return remoteClient.call('vm.suspend', { hostId, vmName })
  }
  return WailsAPI.VMSuspend(hostId, vmName)
}
