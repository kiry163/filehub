---
name: filehub-cli
description: "FileHub CLI for uploading, listing, sharing, downloading, and deleting files. Works with a FileHub server."
metadata:
  {
    "openclaw":
      {
        "emoji": "📦",
        "requires": { "bins": ["filehub-cli"] },
        "install":
          [
            {
              "id": "install-script",
              "kind": "script",
              "url": "https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh",
              "label": "Install FileHub (curl | bash)",
            }
          ],
      },
  }
---

# FileHub CLI Skill 指南

面向 AI Agent 的 FileHub 使用指南（CLI 方式）。包含安装、初始化与命令示例。

## 1. 安装

推荐一键安装（Docker + CLI）：

```bash
curl -fsSL https://raw.githubusercontent.com/kiry163/filehub/main/scripts/install.sh | bash
```

说明：
- 首次运行会生成配置与随机凭据，并启动 FileHub + MinIO
- 再次运行不会覆盖已有配置和数据
- CLI 默认安装到 `/usr/local/bin/filehub-cli`（无权限则回退到 `~/.local/bin`）

## 2. CLI 配置

配置文件路径：`~/.config/filehub-cli/config.yaml`

初始化（本地默认）：

```bash
filehub-cli config init \
  --endpoint http://localhost:8080 \
  --local-key <local_key>
```

`local_key` 从服务端配置中获取：`~/.filehub/config.yaml`。

## 3. 命令示例

### 3.1 版本

```bash
filehub-cli version
```

### 3.2 上传

```bash
filehub-cli upload ./myfile.zip
filehub-cli upload ./folder --recursive
```

输出示例：
```
filehub://aB3dE9kLmN0P
http://localhost:8080/file/aB3dE9kLmN0P
```

### 3.3 列表

```bash
filehub-cli list --limit 10
```

### 3.4 分享链接

```bash
filehub-cli share filehub://aB3dE9kLmN0P
```

### 3.5 下载

```bash
filehub-cli download filehub://aB3dE9kLmN0P --output ./downloads/
```

### 3.6 删除

```bash
filehub-cli delete filehub://aB3dE9kLmN0P
```

### 3.7 备份数据

压缩 `~/.filehub/data` 便于迁移：

```bash
filehub-cli backup
```

说明:
- 自动排除 MinIO 内部数据（`.minio.sys`）

指定数据目录与输出文件：

```bash
filehub-cli backup --dir ~/.filehub/data --output ./filehub-backup.tar.gz
```

## 4. 常见问题

### 4.1 无法连接服务端

- 确认服务端正在运行：
  ```bash
  curl -I http://localhost:8080/health
  ```
- 检查 CLI 配置中的 `endpoint` 是否正确。

### 4.2 local_key 不匹配

- 重新从 `~/.filehub/config.yaml` 获取 `auth.local_key` 并更新 CLI 配置。
