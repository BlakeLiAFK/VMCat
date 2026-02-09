# MVP 初始化

> 创建: 2026-02-08
> 状态: 已完成

## 目标

基于 Wails + Vue3 + TS 初始化 VMCat 项目骨架，实现 Phase 1 MVP 核心功能。

## 步骤

### 阶段一：项目骨架

- [x] 1. Wails 项目初始化 (vue-ts 模板)
- [x] 2. 配置 .gitignore、wails.json
- [x] 3. 安装前端依赖 (tailwindcss, vue-router, pinia, lucide-vue-next, radix-vue)
- [x] 4. 配置前端工具链 (tailwind, 路由, 状态管理, 路径别名)

### 阶段二：后端核心模块

- [x] 5. SQLite 存储层 (internal/store/) - 宿主机 CRUD
- [x] 6. SSH 客户端封装 (internal/ssh/) - 连接/执行/重连/自动查找 key
- [x] 7. VM 管理模块 (internal/vm/) - virsh 命令封装与 XML 解析
- [x] 8. Wails App 绑定层 (app.go) - 暴露 API 给前端

### 阶段三：前端页面

- [x] 9. 布局框架 (侧边栏 + 主内容区)
- [x] 10. 宿主机管理 (添加/编辑/删除/测试连接)
- [x] 11. VM 列表页 (状态展示 + 操作按钮)
- [x] 12. 仪表盘概览页

### 阶段四：验证

- [x] 13. 编译构建测试 (release 9.9MB, 5.5s)
- [x] 14. SSH 连接 10.0.0.2:23 实机验证
- [x] 15. VM 列表和操作验证 (centos + win10)

## 完成标准

- [x] wails dev 可正常启动
- [x] 可添加宿主机并测试 SSH 连接
- [x] 可查看 VM 列表和执行基本操作 (start/shutdown/destroy)
- [x] 界面美观，使用 shadcn 风格组件 + Tailwind CSS

## 修复记录

- port 类型转换：HTML input 返回 string，Go 需要 int，添加 buildPayload() 转换
- SSH key 默认路径：按 ed25519 > rsa > ecdsa 优先级自动查找
