# FileHub 设计文档

**项目名称**: filehub  
**版本**: v1.0.0  
**日期**: 2026-02-08  
**作者**: 89757 & kiry

---

## 1. 项目概述

### 1.1 目标
构建一个轻量级文件管理服务，支持：
- 通过 Web 界面上传/下载/管理文件
- AI Agent（89757）通过 CLI 操作文件
- 单二进制部署，无需复杂配置
- 移动端友好的响应式设计

### 1.2 核心场景
1. **kiry → 89757**: 上传大文件/多文件，通过 `filehub://{key}` 让 89757 处理
2. **89757 → kiry**: 生成文件后，通过链接让 kiry 下载
3. **文件管理**: Web 界面浏览、预览、删除历史文件

### 1.3 技术选型

| 组件 | 技术 | 说明 |
|------|------|------|
| 后端 | Go 1.23.12 + Gin | 标准 REST API，静态文件服务 |
| 前端 | Vue3 + Vite + Element Plus | 打包后内嵌到 Go |
| 数据库 | SQLite3 | 轻量、单文件、易备份 |
| 文件存储 | MinIO | 对象存储，后端代理模式访问 |
| 部署 | 单二进制 + Docker | Go embed 内嵌前端 |

---

## 2. 系统架构

### 2.1 部署架构

```
┌─────────────────────────────────────────────────────────────┐
│                        用户层                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │   浏览器     │  │   手机      │  │   89757 (AI Agent)  │ │
│  │  (Vue3 Web) │  │   (飞书)    │  │  (filehub-cli skill)  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘ │
└─────────┼────────────────┼───────────────────┼────────────┘
          │                │                   │
          └────────────────┼───────────────────┘
                           │ HTTP
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                      FileHub (Go)                           │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  HTTP Router (Gin)                                   │  │
│  │  ├── GET /          → 返回 index.html (Vue 首页)      │  │
│  │  ├── GET /assets/*  → 返回静态资源 (JS/CSS)           │  │
│  │  └── /api/v1/*      → REST API 端点                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                  │
│  ┌───────────────────────┼──────────────────────────────┐  │
│  │                       ↓                                │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │   SQLite    │  │   本地磁盘    │  │   Go Embed   │  │  │
│  │  │  (元数据)    │  │  (文件存储)   │  │  (前端资源)   │  │  │
│  │  │  - 文件信息   │  │  - 实际文件   │  │  - dist/     │  │  │
│  │  │  - 用户数据   │  │  - 上传目录   │  │  - 打包嵌入   │  │  │
│  │  └─────────────┘  └──────────────┘  └──────────────┘  │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Go Embed 内嵌方案

```
前端项目 (Vue)
├── npm run build
│   └── dist/
│       ├── index.html
│       ├── assets/
│       │   ├── index-xxx.js
│       │   └── index-xxx.css
│       └── ...
│
Go 项目
├── main.go
├── web/
│   └── dist/  ← 复制 Vue 打包结果
│
//go:embed web/dist/*
var webFS embed.FS

http.FS(webFS)  →  嵌入到二进制
```

---

## 3. 数据模型

### 3.1 数据库表结构 (SQLite)

```sql
-- 文件表
CREATE TABLE files (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id         VARCHAR(12) UNIQUE NOT NULL,      -- 随机ID (base62 12位)
    original_name   VARCHAR(255) NOT NULL,            -- 原始文件名（仅用于前端显示）
    object_key      VARCHAR(512) NOT NULL,            -- MinIO 对象键 (格式: YYYY-MM-DD/file_id.ext)
    size            BIGINT NOT NULL,                  -- 文件大小(字节)
    mime_type       VARCHAR(100),                     -- 文件类型
    created_by      VARCHAR(64) NOT NULL,             -- 上传者
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata        JSON                              -- 扩展字段
);

-- 操作日志表
CREATE TABLE audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    action      VARCHAR(50) NOT NULL,                 -- upload/download/delete
    file_id     VARCHAR(32),
    actor       VARCHAR(64) NOT NULL,                 -- 操作者
    ip_address  VARCHAR(45),
    status      VARCHAR(20),                          -- success/failure
    message     TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- JWT 刷新令牌表
CREATE TABLE refresh_tokens (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    token        VARCHAR(128) UNIQUE NOT NULL,
    expires_at   DATETIME NOT NULL,
    is_revoked   BOOLEAN DEFAULT FALSE,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_files_created_by ON files(created_by);
CREATE INDEX idx_files_created_at ON files(created_at);
```

### 3.2 MinIO 对象存储规则

**对象键格式**: `{date}/{file_id}.{ext}`

- `date`: 上传日期，格式 `YYYY-MM-DD`
- `file_id`: 12位 base62 随机ID
- `ext`: 文件扩展名（从原始文件名提取，用于 MIME 类型识别）
- **注意**: 原始文件名仅保存在数据库元数据中，不作为对象键的一部分

**示例**:
  - `2026-02-08/aB3dE9kLmN0P.md`
  - `2026-02-07/xYz9Ab2Cd3Ef.png`

**后端代理模式**:
- 所有文件操作（上传/下载/删除）均通过 FileHub API 中转
- FileHub 服务端与 MinIO 通信，客户端不直接访问 MinIO
- 支持视频流式传输（Range 请求）

---

## 4. API 设计

### 4.1 认证方式

- 登录后使用 JWT 访问 API
- **Header**: `Authorization: Bearer {access_token}`
- access token 有效期 24h，refresh token 用于续期
- **本地调用密钥**: `X-Local-Key: {local_key}`
- 认证优先级: `JWT > X-Local-Key`
- CLI 仅使用 `X-Local-Key`，不使用账号密码登录

### 4.2 接口列表

#### 登录
```
POST /api/v1/auth/login
Content-Type: application/json

Request:
{
  "username": "admin",
  "password": "******"
}

说明:
- 账号与密码来自 `config.yaml` 中的 `auth.admin_username` / `auth.admin_password`

Response 200:
{
  "code": 0,
  "data": {
    "access_token": "jwt-xxx",
    "refresh_token": "rt-xxx",
    "expires_in": 86400
  }
}
```

#### 刷新 Token
```
POST /api/v1/auth/refresh
Content-Type: application/json

Request:
{
  "refresh_token": "rt-xxx"
}

Response 200:
{
  "code": 0,
  "data": {
    "access_token": "jwt-xxx",
    "refresh_token": "rt-xxx",
    "expires_in": 86400
  }
}
```

#### 退出登录
```
POST /api/v1/auth/logout
Authorization: Bearer xxx

Response 200:
{
  "code": 0,
  "message": "logged_out"
}
```

#### 文件上传
```
POST /api/v1/files
Content-Type: multipart/form-data
Authorization: Bearer xxx

Request:
  file: (binary)          -- 文件内容

Response 200:
{
  "code": 0,
  "data": {
    "file_id": "aB3dE9kLmN0P",
    "filehub_url": "filehub://aB3dE9kLmN0P",
    "original_name": "design.md",
    "size": 10240,
    "created_at": "2026-02-08T10:30:00Z"
  }
}
```

#### 文件下载
```
GET /api/v1/files/{file_id}/download
Authorization: Bearer xxx

Response: 文件流 (Content-Disposition: attachment)
```

#### 获取文件信息
```
GET /api/v1/files/{file_id}
Authorization: Bearer xxx

Response 200:
{
  "code": 0,
  "data": {
    "file_id": "aB3dE9kLmN0P",
    "original_name": "design.md",
    "size": 10240,
    "mime_type": "text/markdown",
    "filehub_url": "filehub://aB3dE9kLmN0P",
    "created_at": "2026-02-08T10:30:00Z",
    "download_url": "http://223.109.141.179:8080/api/v1/files/aB3dE9kLmN0P/download"
  }
}
```

#### 文件列表
```
GET /api/v1/files?limit=20&offset=0&order=desc
Authorization: Bearer xxx

Response 200:
{
  "code": 0,
  "data": {
    "total": 100,
    "files": [
      {
        "file_id": "aB3dE9kLmN0P",
        "original_name": "design.md",
        "size": 10240,
        "filehub_url": "filehub://aB3dE9kLmN0P",
        "created_at": "2026-02-08T10:30:00Z"
      }
    ]
  }
}
```

#### 删除文件
```
DELETE /api/v1/files/{file_id}
Authorization: Bearer xxx

Response 200:
{
  "code": 0,
  "message": "deleted"
}
```

#### 获取分享链接
```
GET /api/v1/files/{file_id}/share
Authorization: Bearer xxx

Response 200:
{
  "code": 0,
  "data": {
    "url": "http://223.109.141.179:8080/file/aB3dE9kLmN0P"
  }
}
```

### 4.3 响应结构与错误码

**统一响应结构**:
```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

**错误响应示例**:
```json
{
  "code": 10001,
  "message": "unauthorized",
  "data": null
}
```

**错误码约定**:
- `0`: 成功
- `10001`: 未登录或登录失效
- `10002`: 无权限
- `10003`: 资源不存在
- `10004`: 参数错误
- `10005`: 文件上传失败
- `10006`: 文件下载失败
- `10007`: 文件删除失败
- `10008`: 登录失败
- `10009`: refresh token 无效或过期
- `19999`: 服务器内部错误

### 4.4 HTTP 状态码映射

| 场景 | HTTP 状态码 | 说明 |
|------|-------------|------|
| 成功 | 200 | 请求成功 |
| 参数错误 | 400 | 参数校验失败 |
| 未登录 | 401 | access token 缺失或失效 |
| 无权限 | 403 | 权限不足 |
| 资源不存在 | 404 | file_id 不存在 |
| 冲突 | 409 | 资源冲突（例如重复创建） |
| 上传失败 | 422 | 文件上传失败或校验失败 |
| 服务器错误 | 500 | 未捕获异常 |

---

## 5. CLI 工具设计 (filehub-cli)

### 5.1 安装
```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash
```

### 5.2 配置
```bash
filehub-cli config init
# 输入: API endpoint, Local Key, Public endpoint (optional)

# 配置文件: ~/.config/filehub-cli/config.yaml
```

说明:
- CLI 仅使用 `X-Local-Key` 调用，不保存账号密码
- `public_endpoint` 用于拼接可公网访问的分享链接（不填则使用 `endpoint`）
- 如果未填写 `public_endpoint`，CLI 会调用 `https://api.ip.sb/jsonip` 获取公网 IP，并写入 `http://<ip>:<port>`

### 5.3 命令

```bash
# 上传文件
filehub-cli upload /path/to/local/file.pdf
# 输出: filehub://aB3dE9kLmN0P

# 上传并指定显示名称
filehub-cli upload /path/to/file.pdf --name "设计文档.pdf"

# 下载文件
filehub-cli download filehub://aB3dE9kLmN0P --output ./downloads/

# 列出文件
filehub-cli list --limit 10

# 删除文件
filehub-cli delete filehub://aB3dE9kLmN0P

# 获取分享链接
filehub-cli share filehub://aB3dE9kLmN0P
```

### 5.4 Docker 部署下的使用

**CLI 独立运行**：
- `filehub-cli` 安装在宿主机（或任何可访问 filehub API 的机器）
- 通过 HTTP API 与 Docker 容器通信
- 所有操作经过服务端处理，保证数据一致性

```bash
# 宿主机运行 CLI
filehub-cli upload /home/kiry/document.pdf
# → HTTP POST → filehub 容器 → 写数据库 + 存文件 → 返回 filehub://xxx
```

**注意**：请勿直接复制文件到 Docker Volume 挂载目录，这会导致数据库与文件系统不一致。

### 5.5 批量上传支持

如需批量上传，使用 CLI 的批量功能（所有操作仍走 API）：

```bash
# 批量上传多个文件
filehub-cli upload *.pdf *.jpg

# 递归上传目录
filehub-cli upload /path/to/folder --recursive

# 从文件列表上传
filehub-cli upload --list-file files.txt
```

### 5.5 CLI 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        宿主机                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              filehub-cli (Go 二进制)                  │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  命令: upload /home/kiry/document.pdf          │  │  │
│  │  │         ↓                                      │  │  │
│  │  │  读取本地文件 → HTTP POST → filehub API        │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  └──────────────────────────┬───────────────────────────┘  │
│                             │ HTTP API                     │
│              Docker 网络 (bridge/host)                     │
│                             ↓                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              filehub (Docker 容器)                    │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │  │
│  │  │   Gin API    │  │  本地磁盘     │  │  SQLite   │  │  │
│  │  │   (8080)     │  │  /app/data   │  │  元数据    │  │  │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**重要**：所有文件操作必须通过 API，确保数据库与文件系统一致。

---

## 6. Web 界面设计

### 6.1 页面路由

| 路由 | 页面 | 说明 |
|------|------|------|
| `/` | 文件列表 | 首页，展示所有文件 |
| `/upload` | 上传页面 | 拖拽上传、进度显示 |
| `/file/{id}` | 文件详情 | 预览、下载、分享 |
| `/login` | 登录页 | 未登录访问时跳转 |

### 6.2 核心功能

**文件列表页**:
- 按日期分组显示
- 支持搜索（文件名）
- 批量选择/删除
- 点击复制 `filehub://` 链接

**上传页面**:
- 拖拽上传区域
- 上传进度条
- 上传完成后显示 `filehub://` 链接
- 一键复制链接

**文件详情页**:
- 图片：点击放大预览
- 文本：直接显示内容
- 其他：显示下载按钮
- 未登录访问 `/file/{id}` 自动跳转登录，登录后回跳目标文件

### 6.3 响应式设计

- 桌面：侧边栏 + 主内容区
- 平板/手机：底部导航栏 + 全屏内容

### 6.4 视觉风格与设计系统

**风格关键词**:
- 现代、精致、轻拟物玻璃感、克制的高质感

**色彩系统（避免紫色调）**:
- 主色: `#0F766E`（深青）
- 次色: `#06B6D4`（青蓝）
- CTA: `#F59E0B`（暖橙）
- 背景: `#F8FAFC`（冷灰白）
- 面板: `#FFFFFF` / `rgba(255,255,255,0.75)`
- 文字主色: `#0F172A`
- 文字次色: `#475569`
- 边框: `#E2E8F0`

**字体**:
- 标题: `Space Grotesk`
- 正文: `Manrope`
- 字体引入:
```css
@import url('https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;600;700&family=Manrope:wght@400;500;600;700&display=swap');
```

**基础样式 token（建议 CSS 变量）**:
```css
:root {
  --fh-primary: #0F766E;
  --fh-secondary: #06B6D4;
  --fh-cta: #F59E0B;
  --fh-bg: #F8FAFC;
  --fh-surface: #FFFFFF;
  --fh-surface-glass: rgba(255, 255, 255, 0.75);
  --fh-text: #0F172A;
  --fh-muted: #475569;
  --fh-border: #E2E8F0;
  --fh-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
  --fh-radius-lg: 16px;
  --fh-radius-md: 12px;
  --fh-radius-sm: 10px;
}
```

**背景与层次**:
- 背景使用轻微渐变：`linear-gradient(135deg, #F8FAFC 0%, #ECFEFF 100%)`
- 卡片使用轻玻璃质感 + 细边框 + 轻阴影

**图标**:
- 统一使用 Lucide / Heroicons SVG，大小 20/24px

### 6.5 组件与页面布局

**通用布局**:
- 顶部导航（Logo + 搜索 + 账户菜单）
- 桌面左侧栏（Files / Upload / Activity / Settings）
- 主内容区：列表/详情/上传区域

**文件列表（桌面）**:
- 顶部工具栏：搜索框、排序、筛选、批量操作
- 文件项为卡片行：左侧文件类型图标 + 文件名 + 规格（大小、日期），右侧操作按钮（复制 filehub://、下载、删除）
- 日期分组标题突出显示，使用分隔条

**文件列表（移动）**:
- 顶部固定搜索 + 筛选
- 文件卡片纵向排列，操作按钮放入更多菜单
- 底部导航栏固定（Files / Upload / Search / Account）

**上传页**:
- 玻璃感拖拽区域 + 进度条
- 上传完成弹出结果卡片：`filehub://{key}` + 复制按钮

**文件详情页**:
- 顶部信息区：文件名、大小、创建时间、filehub://
- 内容区：图片预览/文本内容/下载按钮
- 右侧操作区（桌面）或底部动作条（移动）

**登录页**:
- 居中卡片 + 背景渐变
- 强调安全与私有文件存储的文案

### 6.6 交互与可用性要求

- 触控目标最小 44x44px
- 所有交互元素具备 hover/active/focus 状态
- 主按钮在异步操作时禁用并显示 loading
- 重要操作（删除）使用二次确认弹窗
- `prefers-reduced-motion` 下关闭或减少动画

### 6.7 页面线框与模块图

**全局组件**:
- 顶部导航栏：Logo、搜索、上传按钮、用户菜单
- 侧边栏（桌面）：Files / Upload / Activity / Settings
- 底部导航（移动）：Files / Upload / Search / Account

**桌面端：文件列表页（/）**:
```
┌──────────────────────────────────────────────────────────────┐
│ Top Nav: Logo | Search | Upload | User                       │
├───────────────┬──────────────────────────────────────────────┤
│ Sidebar       │ Toolbar: Search | Sort | Filter | Batch      │
│ Files         │──────────────────────────────────────────────│
│ Upload        │ Date Group: 2026-02-08                        │
│ Activity      │ ┌──────────────────────────────────────────┐ │
│ Settings      │ │ [Icon] file-name.pdf   12MB  10:30  ...  │ │
│               │ └──────────────────────────────────────────┘ │
│               │ Date Group: 2026-02-07                        │
│               │ ┌──────────────────────────────────────────┐ │
│               │ │ [Icon] photo.jpg       3MB   09:10  ...  │ │
│               │ └──────────────────────────────────────────┘ │
└───────────────┴──────────────────────────────────────────────┘
```

**桌面端：上传页（/upload）**:
```
┌──────────────────────────────────────────────────────────────┐
│ Top Nav                                                     │
├───────────────┬──────────────────────────────────────────────┤
│ Sidebar       │  ┌────────────────────────────────────────┐  │
│               │  │   Drag & Drop / Click to Upload        │  │
│               │  │   Supported: PDF/PNG/TXT ...           │  │
│               │  └────────────────────────────────────────┘  │
│               │  Upload Queue                               │
│               │  [#######-------] 68%  file-name.pdf         │
│               │  Result Card: filehub://aB3dE9kLmN0P [Copy]  │
└───────────────┴──────────────────────────────────────────────┘
```

**桌面端：文件详情页（/file/{id}）**:
```
┌─────────────────────────────────────────────────────────────┐
│                      FileHub (Go)                           │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  HTTP Router (Gin)                                   │  │
│  │  ├── GET /          → 返回 index.html (Vue 首页)      │  │
│  │  ├── GET /assets/*  → 返回静态资源 (JS/CSS)           │  │
│  │  └── /api/v1/*      → REST API 端点                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                  │
│  ┌───────────────────────┼──────────────────────────────┐  │
│  │                       ↓                                │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │   SQLite    │  │    MinIO      │  │   Go Embed   │  │  │
│  │  │  (元数据)    │  │  (对象存储)   │  │  (前端资源)   │  │  │
│  │  │  - 文件信息   │  │  - 实际文件   │  │  - dist/     │  │  │
│  │  │  - 用户数据   │  │  - bucket     │  │  - 打包嵌入   │  │  │
│  │  └─────────────┘  └──────────────┘  └──────────────┘  │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**移动端：文件列表页（/）**:
```
┌─────────────────────────────┐
│ Search + Filter             │
├─────────────────────────────┤
│ 2026-02-08                   │
│ [Icon] file-name.pdf   ...   │
│ [Icon] photo.jpg       ...   │
│ 2026-02-07                   │
│ [Icon] doc.txt         ...   │
├─────────────────────────────┤
│ Bottom Nav: Files Upload ... │
└─────────────────────────────┘
```

**移动端：上传页（/upload）**:
```
┌─────────────────────────────┐
│ Drag Area / Upload Button   │
│ Progress List               │
│ Result: filehub://... [Copy]│
├─────────────────────────────┤
│ Bottom Nav                  │
└─────────────────────────────┘
```

**移动端：文件详情页（/file/{id}）**:
```
┌─────────────────────────────┐
│ file-name.pdf               │
│ filehub://... [Copy]        │
│ Preview                     │
│ Actions: Download / Delete  │
├─────────────────────────────┤
│ Bottom Nav                  │
└─────────────────────────────┘
```

**登录页（/login）**:
```
┌────────────────────────────────────────────┐
│ Logo + Title                               │
│ Username [______________]                  │
│ Password [______________]                  │
│ [ Login ]                                  │
│ Tip: Secure private storage                │
└────────────────────────────────────────────┘
```

---

## 7. 项目结构

```
filehub/
├── cmd/
│   └── filehub/
│       └── main.go           # 入口
├── internal/
│   ├── api/                  # HTTP 路由和处理器
│   ├── service/              # 业务逻辑
│   ├── storage/              # 文件存储
│   ├── db/                   # 数据库
│   └── config/               # 配置管理
├── web/
│   └── dist/                 # Vue 打包结果 (embed)
├── web-ui/                   # Vue 前端项目
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── vite.config.js
├── scripts/
│   └── install.sh            # CLI 安装脚本
├── data/                     # 运行时数据 (gitignore)
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

---

## 8. 部署方案

### 8.1 开发模式

```bash
# 1. 启动后端
cd filehub
go run cmd/filehub/main.go

# 2. 启动前端（单独）
cd web-ui
npm install
npm run dev
```

### 8.2 生产模式（单二进制）

```bash
# 1. 构建前端
cd web-ui
npm install
npm run build

# 2. 复制到 Go 项目
cp -r dist/ ../web/

# 3. 构建 Go 二进制
cd ..
go build -o filehub cmd/filehub/main.go

# 4. 运行
./filehub
```

### 8.3 Docker 部署

### 8.3.1 一键安装（Docker + CLI）

> 适用于 Linux/amd64，使用 GHCR 公共镜像。

```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash
```

默认行为:
- 安装目录: `~/.filehub`
- 启动服务: FileHub(8080) + MinIO(9000/9001)
- 自动生成并写入 `config.yaml`（包含 admin 密码与 local_key）
- 安装 CLI 到 `~/.local/bin/filehub-cli` 并初始化配置

可选参数示例:
```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash -s -- --port 18080 --version v0.1.0
```

说明:
- 默认会调用 `https://api.ip.sb/jsonip` 探测公网 IP，并将 CLI 的 `public_endpoint` 初始化为 `http://<ip>:<port>`
- 如需强制使用本地地址，可指定：
```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash -s -- --public-endpoint http://localhost:8080
```

```dockerfile
# 多阶段构建
FROM node:18-alpine AS web-builder
WORKDIR /app/web-ui
COPY web-ui/package*.json ./
RUN npm ci
COPY web-ui/ ./
RUN npm run build

FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web-ui/dist ./web/dist
RUN go build -o filehub cmd/filehub/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=go-builder /app/filehub .
EXPOSE 8080
CMD ["./filehub"]
```

```bash
# 构建并运行
docker build -t filehub .
docker run -p 8080:8080 -v $(pwd)/data:/data filehub
```

---

## 8.4 配置项清单

**配置文件**: `config.yaml`

```yaml
# 服务端配置
server:
  port: 8080                          # 服务监听端口
  log_level: info                     # 日志级别 (debug/info/warn/error)

# 数据库配置
database:
  path: ./data/filehub.db             # SQLite 数据库路径

# JWT 认证配置
auth:
  jwt_secret: "your-secret-key"       # JWT 签名密钥（必填，建议 ≥32 字符）
  jwt_expire_hours: 24                # access token 有效期（小时）
  refresh_expire_days: 7              # refresh token 有效期（天）
  admin_username: admin               # 初始化管理员用户名
  admin_password: "your-password"     # 初始化管理员密码（必填）
  local_key: "your-local-key"         # 本地调用密钥（Agent 使用）

# 文件上传限制
upload:
  max_size_mb: 1024                   # 单文件最大上传大小（MB）

# MinIO 对象存储配置
minio:
  endpoint: localhost:9000            # MinIO 服务端点
  access_key: "your-access-key"       # Access Key
  secret_key: "your-secret-key"       # Secret Key
  bucket: filehub                     # Bucket 名称
  use_ssl: false                      # 是否使用 SSL
  region: ""                          # 区域（可选，默认 us-east-1）
```

**配置加载优先级**:
1. 默认配置（代码中定义）
2. 配置文件 `config.yaml`
3. 环境变量（覆盖配置文件）

**环境变量命名规则**: `FILEHUB_<section>_<key>`，例如：
- `FILEHUB_SERVER_PORT=9090`
- `FILEHUB_MINIO_ENDPOINT=minio.example.com:9000`
- `FILEHUB_AUTH_JWT_SECRET=xxx`
- `FILEHUB_AUTH_LOCAL_KEY=xxx`

---

## 9. 安全考虑

1. **JWT**: 前端登录后使用 access token，refresh token 续期
2. **管理员凭据**: 账号与密码写入配置文件，仅用于 Web 登录
3. **本地调用密钥**: `X-Local-Key` 仅用于 CLI/Agent 调用（所有 API 可用）
4. **分享链接**: 不含 token，依赖登录态控制访问
5. **文件类型限制**: 可配置禁止上传可执行文件
6. **文件大小限制**: 可配置单文件最大限制
7. **审计日志**: 所有操作记录日志，便于追溯

### 9.1 审计日志字段规范

**字段说明**:
- `action`: upload/download/delete/share/login/logout/refresh
- `file_id`: 关联文件（可为空）
- `actor`: 操作人（单用户可固定为 admin）
- `ip_address`: 客户端 IP
- `status`: success/failure
- `message`: 失败原因或补充说明
- `created_at`: 发生时间

**建议扩展字段**:
- `user_agent`: 客户端 UA
- `request_id`: 请求链路追踪 ID
- `size`: 上传/下载字节数

---

## 10. 开发计划

### Phase 1: MVP (MinIO + 核心功能)
- [x] 后端 API 骨架 (Go + Gin + SQLite)
- [x] MinIO 对象存储集成
- [ ] YAML 配置文件支持
- [ ] 审计日志写入
- [ ] 视频流 Range 支持
- [ ] 文件名搜索

### Phase 2: 前端与部署
- [ ] Vue3 前端开发
- [ ] Go embed 静态资源内嵌
- [ ] Docker 多阶段构建
- [ ] 单二进制部署测试

### Phase 3: CLI 与集成
- [ ] filehub-cli 工具开发
- [ ] 89757 AI Agent 集成
- [ ] 批量上传支持
- [ ] 端到端测试

### Phase 4: 增强功能 (后续)
- [ ] 分片上传/断点续传
- [ ] 图片缩略图生成
- [ ] 高级搜索与标签
- [ ] 多用户支持

---

## 11. 参考资源

- [Gin Web Framework](https://gin-gonic.com/)
- [Vue3 Documentation](https://vuejs.org/)
- [Go Embed](https://pkg.go.dev/embed)
- [Element Plus](https://element-plus.org/)

---

**End of Document**
