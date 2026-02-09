# VMCat - 轻量级虚拟机管理工具

> 创建: 2026-02-08
> 状态: 进行中

## 一、项目概述

VMCat 是一个基于 Go + Wails 的桌面应用，通过 SSH 连接远程宿主机，管理 libvirt/KVM 虚拟机。
核心理念：**只需配置宿主机 SSH，即可完成所有管理操作，无需在宿主机部署任何额外组件。**

### 核心价值

- 零部署：不需要在宿主机安装 agent，SSH 即用
- 多机管理：统一管理多台 KVM 宿主机
- 内嵌控制台：浏览器级 VNC 体验，无需额外 VNC 客户端
- 轻量级：单个二进制文件，跨平台 (macOS/Windows/Linux)

---

## 二、技术栈

| 层级 | 技术选型 | 说明 |
|------|---------|------|
| 桌面框架 | Wails v2 | Go + Web 前端，编译为原生桌面应用 |
| 后端语言 | Go 1.22+ | 主要业务逻辑 |
| 前端框架 | Vue 3 + TypeScript | UI 层 |
| UI 组件库 | Shadcn-vue | 现代化组件，支持暗色主题 |
| SSH | golang.org/x/crypto/ssh | SSH 连接、命令执行、端口转发 |
| VNC 显示 | noVNC | WebSocket → VNC，内嵌到前端 |
| 本地存储 | SQLite (go-sqlite3) | 存储宿主机配置、凭据等 |
| 构建工具 | Vite | 前端构建 |

---

## 三、系统架构

```
+----------------------------------------------------------+
|                     VMCat 桌面应用                         |
|                                                          |
|  +-------------------+    +---------------------------+  |
|  |   Vue3 前端        |    |      Go 后端              |  |
|  |                   |    |                           |  |
|  | +- Dashboard     |    |  +- HostManager           |  |
|  | +- HostList      |<-->|  |  管理宿主机连接配置       |  |
|  | +- VMList        |    |  |                         |  |
|  | +- VMDetail      |    |  +- SSHClient              |  |
|  | +- VNCConsole    |    |  |  SSH连接池/命令执行/隧道   |  |
|  | +- Settings      |    |  |                         |  |
|  |                   |    |  +- VMController          |  |
|  | (noVNC 组件)      |    |  |  virsh命令封装           |  |
|  |   |               |    |  |                         |  |
|  +---+---------------+    |  +- VNCProxy              |  |
|      |                    |  |  SSH隧道+WebSocket代理   |  |
|      | WebSocket          |  |                         |  |
|      +---------------------->+- MonitorService         |  |
|                           |  |  资源采集(CPU/MEM/IO)    |  |
|                           |  |                         |  |
|                           |  +- Store (SQLite)         |  |
|                           |     本地数据持久化           |  |
|                           +---------------------------+  |
+----------------------------------------------------------+
       |  SSH (每台宿主机一个连接)
       v
+----------------+  +----------------+  +----------------+
|  宿主机 A       |  |  宿主机 B       |  |  宿主机 C       |
|  libvirtd      |  |  libvirtd      |  |  libvirtd      |
|  +----+ +----+ |  |  +----+ +----+ |  |  +----+ +----+ |
|  | VM | | VM | |  |  | VM | | VM | |  |  | VM | | VM | |
|  +----+ +----+ |  |  +----+ +----+ |  |  +----+ +----+ |
+----------------+  +----------------+  +----------------+
```

---

## 四、核心模块设计

### 4.1 SSH 连接管理 (`internal/ssh/`)

负责与宿主机的所有通信。

```go
// 连接配置
type HostConfig struct {
    ID         string // 唯一标识
    Name       string // 显示名称
    Host       string // 地址
    Port       int    // SSH 端口
    User       string // 用户名
    AuthMethod string // "key" | "password"
    KeyPath    string // 私钥路径 (AuthMethod=key 时)
    Password   string // 密码 (加密存储)
    ProxyJump  string // 跳板机 (可选)
}

// SSH 客户端接口
type Client interface {
    Connect(cfg HostConfig) error
    Execute(cmd string) (string, error)
    Forward(localPort, remoteHost string, remotePort int) error
    Close() error
    IsAlive() bool
}
```

**关键设计：**

- **连接池**：每台宿主机维护一个 SSH 连接，多个操作复用同一连接
- **自动重连**：连接断开后自动重连，对上层透明
- **ProxyJump 支持**：支持通过跳板机连接（适配复杂网络环境）
- **并发安全**：SSH Session 不支持并发，需要 mutex 或 session 池

### 4.2 VM 管理 (`internal/vm/`)

封装 virsh 命令，提供结构化 API。

```go
// 虚拟机数据模型
type VM struct {
    ID        int
    Name      string
    State     string   // running | shut off | paused | ...
    CPUs      int
    MemoryMB  int
    Autostart bool
    DiskPaths []string
    NICs      []NIC
    VNCPort   int
    HostID    string   // 所属宿主机
}

type NIC struct {
    MAC    string
    Bridge string
    IP     string
    Model  string
}

// VM 管理接口
type Manager interface {
    // 查询
    List(hostID string) ([]VM, error)
    Get(hostID, vmName string) (*VM, error)
    GetStats(hostID, vmName string) (*VMStats, error)

    // 生命周期
    Start(hostID, vmName string) error
    Shutdown(hostID, vmName string) error    // 优雅关机 (ACPI)
    Destroy(hostID, vmName string) error     // 强制关机
    Reboot(hostID, vmName string) error
    Suspend(hostID, vmName string) error
    Resume(hostID, vmName string) error

    // 配置
    SetAutostart(hostID, vmName string, enabled bool) error
    SetVCPUs(hostID, vmName string, count int) error
    SetMemory(hostID, vmName string, sizeMB int) error

    // 快照
    SnapshotCreate(hostID, vmName, snapName string) error
    SnapshotList(hostID, vmName string) ([]Snapshot, error)
    SnapshotRevert(hostID, vmName, snapName string) error
    SnapshotDelete(hostID, vmName, snapName string) error
}
```

**virsh 命令映射：**

| 操作 | virsh 命令 | 输出解析 |
|------|-----------|---------|
| 列表 | `virsh list --all` | 按行拆分，解析 ID/Name/State |
| 详情 | `virsh dominfo <name>` | Key-Value 解析 |
| 网卡IP | `virsh domifaddr <name>` | 表格解析 |
| VNC端口 | `virsh vncdisplay <name>` | 直接取端口号 |
| 资源统计 | `virsh domstats <name>` | Key-Value 解析 |
| XML详情 | `virsh dumpxml <name>` | XML 解析 (encoding/xml) |
| 启动 | `virsh start <name>` | 检查返回码 |
| 关机 | `virsh shutdown <name>` | 检查返回码 |
| 快照 | `virsh snapshot-create-as <name> --name <snap>` | 检查返回码 |

**输出解析策略：**

优先用 `virsh dumpxml` 获取 XML 结构化数据，辅以其他命令补充运行时信息。
XML 解析比文本解析稳定可靠，不受 locale 和版本影响。

### 4.3 VNC 控制台 (`internal/vnc/`)

这是体验关键，架构：

```
前端 noVNC <--WebSocket--> Go VNC Proxy <--SSH Tunnel--> 宿主机:VNC端口
```

**流程：**

1. 用户点击「打开控制台」
2. Go 后端通过 SSH 建立端口转发：`本地随机端口 → 宿主机 127.0.0.1:VNC端口`
3. Go 后端启动 WebSocket 服务，监听另一个本地端口
4. WebSocket 服务将流量桥接到 SSH 隧道的本地端口
5. 前端 noVNC 连接 WebSocket 地址

```go
// VNC 代理
type Proxy struct {
    sshClient *ssh.Client
    wsPort    int // 本地 WebSocket 监听端口
}

func (p *Proxy) Start(vncHost string, vncPort int) (wsURL string, err error) {
    // 1. SSH 端口转发
    localPort := getFreePort()
    go p.sshClient.Forward(localPort, vncHost, vncPort)

    // 2. WebSocket → TCP 桥接
    p.wsPort = getFreePort()
    go p.bridgeWebSocketToTCP(p.wsPort, localPort)

    return fmt.Sprintf("ws://127.0.0.1:%d", p.wsPort), nil
}
```

**noVNC 集成：**

- 将 noVNC 作为前端静态资源打包
- 通过 iframe 或直接引入 noVNC 的 RFB 类
- 支持剪贴板同步、分辨率自适应、全屏模式

### 4.4 宿主机监控 (`internal/monitor/`)

通过 SSH 执行系统命令采集数据，定时轮询。

```go
type HostStats struct {
    CPUUsage    float64   // 百分比
    MemTotal    int64     // bytes
    MemUsed     int64
    DiskTotal   int64
    DiskUsed    int64
    LoadAvg     [3]float64
    Uptime      string
    VMCount     int       // 运行中 VM 数量
    CollectedAt time.Time
}
```

**采集命令：**

| 指标 | 命令 |
|------|------|
| CPU | `top -bn1 | grep Cpu` 或 `/proc/stat` 两次采样计算 |
| 内存 | `free -b` |
| 磁盘 | `df -B1 /` |
| 负载 | `cat /proc/loadavg` |
| 运行时间 | `uptime -p` |

**轮询策略：**
- 默认 10 秒一次，可配置
- 只在界面可见时轮询，切换到其他页面暂停
- 前端通过 Wails Events 接收推送

### 4.5 本地存储 (`internal/store/`)

SQLite 存储宿主机配置和应用设置。

```sql
-- 宿主机
CREATE TABLE hosts (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    host        TEXT NOT NULL,
    port        INTEGER DEFAULT 22,
    user        TEXT DEFAULT 'root',
    auth_method TEXT DEFAULT 'key',     -- key | password
    key_path    TEXT,
    password    TEXT,                    -- AES 加密存储
    proxy_jump  TEXT,
    sort_order  INTEGER DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 设置
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT
);
```

**安全考虑：**
- 密码使用 AES-256-GCM 加密存储
- 密钥派生自机器唯一标识 (macOS: IOPlatformUUID)
- 优先使用 SSH Key 认证，密码作为备选

---

## 五、前端页面设计

### 5.1 页面结构

```
+-- 侧边栏 (固定) --+------------- 主内容区 ---------------+
|                    |                                      |
|  VMCat Logo        |  [面包屑导航]                         |
|                    |                                      |
|  -- 宿主机列表 --   |  根据侧边栏选择显示不同内容：           |
|  > homelab    (3)  |                                      |
|    t0.1kb.win (2)  |  - 仪表盘: 所有宿主机概览              |
|    t4.1kb.win (1)  |  - 宿主机详情: VM列表 + 资源监控        |
|                    |  - VM详情: 配置 + 操作 + 快照           |
|  -- 工具 --         |  - 控制台: noVNC 全屏                  |
|  + 添加宿主机       |  - 设置: 全局配置                      |
|  ⚙ 设置             |                                      |
|                    |                                      |
+--------------------+--------------------------------------+
```

### 5.2 页面清单

| 页面 | 路由 | 功能 |
|------|------|------|
| 仪表盘 | `/` | 所有宿主机状态概览，VM总数/运行数 |
| 宿主机详情 | `/host/:id` | VM列表、宿主机资源图表、操作按钮 |
| VM 详情 | `/host/:id/vm/:name` | VM配置信息、资源统计、快照管理 |
| VNC 控制台 | `/host/:id/vm/:name/console` | noVNC 嵌入，支持全屏 |
| 添加宿主机 | `/host/new` | 表单：地址、端口、认证方式、测试连接 |
| 设置 | `/settings` | 全局偏好设置 |

### 5.3 核心交互

**VM 列表页：**

```
宿主机: homelab (10.0.0.2:23)          CPU: 45%  MEM: 12.3/15.6 GB
+-------+----------+--------+------+------+---+--------------------+
| 状态   | 名称     | CPU    | 内存  | IP   |   | 操作               |
+-------+----------+--------+------+------+---+--------------------+
| ● 运行 | centos   | 4 vCPU | 8 GB | .113 |   | [控制台] [关机] [...] |
| ● 运行 | win10    | 4 vCPU | 8 GB | .56  |   | [控制台] [关机] [...] |
| ○ 关机 | ubuntu   | 2 vCPU | 4 GB | -    |   | [启动] [...]        |
+-------+----------+--------+------+------+---+--------------------+
```

**VNC 控制台页：**
- 顶部工具栏：发送 Ctrl+Alt+Del、全屏、截图、缩放
- 主体区域：noVNC 画面，自适应窗口大小
- 底部状态栏：连接状态、延迟

---

## 六、项目目录结构

```
vmcat/
├── main.go                         # Wails 入口
├── app.go                          # Wails App 绑定层
├── wails.json                      # Wails 配置
├── go.mod
├── go.sum
│
├── internal/
│   ├── ssh/
│   │   ├── client.go               # SSH 客户端封装
│   │   ├── pool.go                 # 连接池管理
│   │   └── tunnel.go               # SSH 端口转发
│   │
│   ├── vm/
│   │   ├── manager.go              # VM 管理器 (业务逻辑)
│   │   ├── virsh.go                # virsh 命令执行与解析
│   │   └── model.go                # 数据模型
│   │
│   ├── vnc/
│   │   └── proxy.go                # WebSocket ←→ VNC 桥接
│   │
│   ├── monitor/
│   │   ├── collector.go            # 宿主机指标采集
│   │   └── model.go                # 监控数据模型
│   │
│   └── store/
│       ├── db.go                   # SQLite 初始化与迁移
│       └── host.go                 # 宿主机 CRUD
│
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.ts
│       ├── App.vue
│       ├── router/
│       │   └── index.ts            # 路由定义
│       ├── views/
│       │   ├── Dashboard.vue       # 仪表盘
│       │   ├── HostDetail.vue      # 宿主机详情 + VM 列表
│       │   ├── VMDetail.vue        # VM 详情 + 快照
│       │   ├── VNCConsole.vue      # noVNC 控制台
│       │   ├── HostForm.vue        # 添加/编辑宿主机
│       │   └── Settings.vue        # 设置
│       ├── components/
│       │   ├── Sidebar.vue         # 侧边栏
│       │   ├── VMTable.vue         # VM 列表表格
│       │   ├── StatsCard.vue       # 资源统计卡片
│       │   └── SnapshotList.vue    # 快照列表
│       ├── composables/
│       │   ├── useHost.ts          # 宿主机相关逻辑
│       │   ├── useVM.ts            # VM 相关逻辑
│       │   └── useMonitor.ts       # 监控数据订阅
│       ├── stores/
│       │   └── app.ts              # Pinia 全局状态
│       └── assets/
│           └── styles/
│               └── main.css        # Tailwind + 自定义样式
│
└── plans/                          # 项目计划文档
    ├── STATUS.md
    ├── active/
    ├── completed/
    └── docs/
```

---

## 七、Wails 绑定层 API

Go 后端暴露给前端的方法（通过 Wails Bind）：

```go
type App struct {
    sshPool    *ssh.Pool
    vmManager  *vm.Manager
    vncProxy   *vnc.Proxy
    monitor    *monitor.Service
    store      *store.Store
}

// === 宿主机管理 ===
func (a *App) HostList() ([]Host, error)
func (a *App) HostAdd(cfg HostConfig) error
func (a *App) HostUpdate(cfg HostConfig) error
func (a *App) HostDelete(id string) error
func (a *App) HostTest(cfg HostConfig) error      // 测试连接

// === VM 管理 ===
func (a *App) VMList(hostID string) ([]VM, error)
func (a *App) VMGet(hostID, name string) (*VM, error)
func (a *App) VMStart(hostID, name string) error
func (a *App) VMShutdown(hostID, name string) error
func (a *App) VMDestroy(hostID, name string) error
func (a *App) VMReboot(hostID, name string) error
func (a *App) VMSetAutostart(hostID, name string, on bool) error

// === 快照 ===
func (a *App) SnapshotList(hostID, vmName string) ([]Snapshot, error)
func (a *App) SnapshotCreate(hostID, vmName, snapName string) error
func (a *App) SnapshotRevert(hostID, vmName, snapName string) error
func (a *App) SnapshotDelete(hostID, vmName, snapName string) error

// === VNC 控制台 ===
func (a *App) VNCConnect(hostID, vmName string) (string, error)  // 返回 ws:// URL
func (a *App) VNCDisconnect(hostID, vmName string) error

// === 监控 ===
func (a *App) HostStats(hostID string) (*HostStats, error)
func (a *App) VMStats(hostID, vmName string) (*VMStats, error)
```

前端调用示例（自动生成 TypeScript 类型）：

```typescript
import { VMList, VMStart, VNCConnect } from '../../wailsjs/go/main/App'

// 获取 VM 列表
const vms = await VMList('homelab')

// 启动 VM
await VMStart('homelab', 'win10')

// 打开 VNC 控制台
const wsURL = await VNCConnect('homelab', 'win10')
// wsURL = "ws://127.0.0.1:38291"
// 传给 noVNC 组件
```

---

## 八、实现路线

### Phase 1 - 基础框架（MVP）

- [ ] Wails 项目初始化 (Go + Vue3 + TS + Shadcn)
- [ ] SQLite 存储层 (宿主机 CRUD)
- [ ] SSH 客户端封装 (连接/执行/重连)
- [ ] 宿主机管理页面 (添加/编辑/删除/测试连接)
- [ ] virsh 命令封装 (list/start/shutdown/destroy)
- [ ] VM 列表页面 (状态、基本操作按钮)

### Phase 2 - VNC 控制台

- [ ] SSH 端口转发
- [ ] WebSocket VNC Proxy
- [ ] noVNC 前端集成
- [ ] 控制台页面 (工具栏 + 全屏 + 自适应)

### Phase 3 - 监控与快照

- [ ] 宿主机资源采集 (CPU/MEM/Disk)
- [ ] 仪表盘概览页
- [ ] 资源图表 (实时曲线)
- [ ] 快照管理 (创建/恢复/删除)
- [ ] VM 详情页完善

### Phase 4 - 体验优化

- [ ] 暗色/亮色主题切换
- [ ] 多语言支持
- [ ] SSH Key 管理 (内置密钥生成)
- [ ] 通知系统 (VM 状态变更提醒)
- [ ] 导入/导出宿主机配置

---

## 九、技术风险与应对

| 风险 | 影响 | 应对策略 |
|------|------|---------|
| virsh 输出格式因版本不同 | 解析失败 | 优先用 `dumpxml` (XML)，文本解析做兜底 |
| SSH 连接不稳定 | 操作中断 | 自动重连 + 操作重试 + 超时控制 |
| noVNC 延迟高 | 控制台卡顿 | 调整编码质量、支持压缩、显示延迟指标 |
| 多宿主机并发操作 | 资源竞争 | 每台宿主机独立 goroutine + 连接池 |
| SQLite 密码存储安全 | 凭据泄露 | AES-256-GCM 加密，密钥绑定机器 |

---

## 十、备注

### 为什么不用 libvirt-go？

- libvirt-go 依赖 CGO + libvirt C 库
- 交叉编译困难（macOS 编译 Linux 版本需要额外工具链）
- 直接解析 virsh 命令输出虽然「笨」但零依赖、跨平台简单
- virsh 是 libvirt 的官方 CLI，输出格式相对稳定

### 为什么不用 Proxmox / Cockpit？

- Proxmox 太重，是完整虚拟化平台，改造现有环境成本高
- Cockpit 是 Web 应用需要在每台宿主机部署，不符合「零部署」理念
- VMCat 定位是轻量桌面工具，针对个人/小团队管理少量宿主机

### 后续演进方向

如果管理规模增大（5+ 台宿主机），可以演进为 Phase 2 架构：
- 宿主机部署轻量 Agent（单个 Go 二进制）
- gRPC 通信替代 SSH 命令解析
- Agent 直接调用 libvirt Go 绑定
- 支持实时事件推送（VM 状态变更、告警）
