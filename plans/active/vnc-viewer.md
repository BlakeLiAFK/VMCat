# VNC 远程桌面集成

> 创建: 2026-02-09
> 状态: 进行中

## 目标

在 VMCat 中直接打开 VM 的 VNC 远程桌面，无需外部 VNC 客户端。

## 架构

```
noVNC (前端) → WebSocket (localhost) → Go 代理 → SSH 隧道 → VNC (KVM宿主机)
```

## 步骤

- [x] 1. 安装 @novnc/novnc 前端依赖 (v1.5.0, 1.6.0 有 Vite 兼容问题)
- [x] 2. 在 terminal server 添加 /ws/vnc 路由（复用端口）
- [x] 3. 创建 VNCViewer.vue 前端页面
- [x] 4. 添加路由 /host/:id/vm/:name/vnc
- [x] 5. VMDetail 页面添加 VNC 按钮
- [x] 6. 构建验证 (通过, 5.9s)
- [ ] 7. 实机测试 (待用户验证)

## 修改文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/terminal/server.go` | 修改 | 添加 handleVNC 处理函数 |
| `frontend/src/views/VNCViewer.vue` | 新建 | noVNC 封装页面 |
| `frontend/src/views/VMDetail.vue` | 修改 | 添加 VNC 按钮 |
| `frontend/src/router/index.ts` | 修改 | 添加 VNC 路由 |
| `frontend/vite.config.ts` | 修改 | 添加 esnext target |

## 功能

- 全屏切换
- 缩放模式切换 (缩放适配 / 原始分辨率)
- Ctrl+Alt+Del 发送
- 自动重连
- 连接状态指示

## 完成标准

- [x] 编译构建通过
- [ ] VNC 连接成功，可看到 VM 桌面
- [ ] 支持键盘鼠标操作
