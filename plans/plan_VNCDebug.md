# VNC Debug Plan

## 问题

Win10 节点 VNC 无法连接

## 分析

### VNC 连接链路

1. **前端** `VNCViewer.vue` -> 创建 noVNC RFB 连接到 `ws://127.0.0.1:{port}/ws/vnc`
2. **后端** `terminal/server.go` `handleVNC` -> 接收 WebSocket，通过 SSH 隧道 dial 到远程 VNC 端口
3. **VNC端口检测** `vm/virsh.go` -> 解析 `virsh dumpxml` XML 获取 `<graphics type="vnc" port="..." />`

### 发现的问题

#### BUG-1: RFB 未导入 (致命)

`VNCViewer.vue` 第9行注释说"noVNC 使用动态导入避免顶层加载问题"，但实际上 **没有任何 import 语句**。
`new RFB(...)` 会抛出 `ReferenceError: RFB is not defined`，导致 VNC 完全无法工作。

#### BUG-2: 后端日志不足

`handleVNC` 中缺少关键环节的调试日志，无法定位 SSH 隧道/VNC 端口问题。

#### BUG-3: 前端错误信息不够详细

disconnect 事件缺少详细信息，无法区分是网络问题还是协议问题。

## 修复内容 (已完成)

### Round 1: 基础修复

#### 1. 前端 VNCViewer.vue

- [x] 添加 `const { default: RFB } = await import('@novnc/novnc/lib/rfb.js')` 动态导入
- [x] 添加 noVNC 类型声明到 `vite-env.d.ts`
- [x] 增强全链路日志: SSH 检查、VM 详情、端口获取、RFB 加载、连接/断开/认证失败

#### 2. 后端 terminal/server.go handleVNC

- [x] 请求参数日志
- [x] SSH dial 逐地址尝试日志
- [x] VNC 握手验证(读取前12字节确认 RFB 协议)
- [x] 握手数据先发给 WebSocket 再启动桥接
- [x] 数据流量统计 (bytesFromVNC/bytesToVNC)
- [x] 双向桥接错误日志
- [x] 代理关闭时输出流量统计

### Round 2: VNC 密码认证 (根因)

通过日志分析确认根因:

- `fromVNC=18` = 2(安全类型头) + 16(VNC auth challenge)
- `toVNC=13` = 12(客户端版本) + 1(选择安全类型2)
- Win10 VM 的 QEMU VNC 启用了密码认证(type 2)，noVNC 触发 credentialsrequired 但前端未处理

#### 修复:

- [x] 处理 `credentialsrequired` 事件，弹出密码输入框
- [x] 添加 `submitPassword()` 调用 `rfb.sendCredentials({ password })`
- [x] 密码弹窗 UI: Lock 图标 + 密码输入 + 回车提交 + 取消/连接按钮
- [x] `vite-env.d.ts` 补充 `sendCredentials` 类型声明

### 自测

- [x] Go 编译通过 (`go build ./...`)
- [x] 前端 TS 类型检查通过 (`vue-tsc --noEmit`)
