# 全功能增强

> 创建: 2026-02-08
> 状态: 进行中

## 目标

在 MVP 基础上完成全部功能增强，使应用达到日常可用级别。

## 步骤

### Phase A - 体验基础升级

- [x] 1. Toast 通知系统 (vue-sonner 替代 alert)
- [x] 2. 暗色主题切换 (useTheme composable + CSS vars)
- [x] 3. VM 列表自动刷新 + 操作后自动刷新 (10秒间隔)
- [x] 4. 侧边栏连接状态自动刷新 (15秒间隔)
- [x] 5. 点击宿主机时自动连接

### Phase B - 数据增强

- [x] 6. 宿主机资源概览 (CPU/MEM/Disk/VM 顶部卡片 + 进度条)
- [x] 7. VM 列表增加 IP 列 (通过 NIC 信息展示)
- [x] 8. 快照管理 (创建/恢复/删除，VMDetail 页面)

### Phase C - 核心高级功能

- [x] 9. SSH 终端 (xterm.js + WebSocket, 自动 fit + resize)
- [x] 10. VNC 连接信息展示 (VMDetail 中显示 VNC 端口)

### Phase D - 运维增强

- [x] 11. 批量操作 (多选 VM, 批量启动/关机)
- [x] 12. 宿主机导入导出 (JSON 格式，不含密码)
- [x] 13. 设置页面 (主题切换 + 数据管理 + 关于)

## 实现方案

### 后端新增模块
- `internal/monitor/monitor.go` - 宿主机资源采集 (单次 SSH 合并命令)
- `internal/vm/snapshot.go` - 快照 CRUD (virsh snapshot-list/create-as/delete/revert)
- `internal/store/settings.go` - 设置项 KV (SQLite)
- `internal/terminal/server.go` - WebSocket 终端服务 (gorilla/websocket, 随机端口)
- `internal/ssh/client.go` - 新增 OpenShell (PTY + Shell)

### 前端改写
- Sidebar / HostDetail / VMDetail / Dashboard 全面升级
- 新增: SSHTerminal.vue, Settings.vue
- 路由新增: /host/:id/terminal, /settings

## 完成标准

- [x] 所有功能可正常使用
- [x] 暗色/亮色主题切换正常
- [x] SSH 终端可连接
- [x] 实机验证通过 (10.0.0.2:23, 2台 VM 正常显示, 资源监控正常)

## 验证结果

- 宿主机自动连接: 通过
- 资源概览: CPU 20.9%, 内存 63.8%, 磁盘 17.5%, VM 2/2
- VM 列表: centos (running), win10 (running) 正确显示
- 批量选择和操作按钮: UI 正常
- 编译构建: 通过 (6.8s)
