# 阿里云 OSS 双 Bucket 可视化文件管理工作台 (OSS Sync Master)

一款使用 **Go (Golang)** 开发的轻量级、跨平台**双 Bucket 可视化文件管理工作台**。专为前端、运营与研发人员设计，解决在两套 OSS Bucket（如测试 ⇋ 生产、源端 ⇋ 目标端）之间重复上传、覆盖与删除静态文件（图片、资源、文档）的痛点。

---

## 🌟 核心特性与工作流

1. **直接进入文件管理工作台**
   - 启动时配置完两端 Bucket（AK/SK/Endpoint/BucketName）及要同步的目录后，**直接进入文件管理主页面**。
   - 无论何时，均可通过右上角「⚙️ 存储桶设置」修改凭证或追加同步目录。

2. **严格目录白名单隔离**
   - 根目录下**仅展示您在配置中指定的同步目录**（例如 `images/`、`static/`）。
   - 任何未配置的根目录及外部文件绝对不可见、不可进入，杜绝误操作与数据越界。

3. **拖拽上传与同名覆盖（双端实时并发写入）**
   - 支持直接从电脑资源管理器中将**一张或多张图片/文件拖拽到页面任意区域**。
   - 拖拽后自动并发上传写入 **Bucket A 与 Bucket B 的同名路径**；若同名文件已存在，直接覆盖更新。
   - 亦可点击右上角「⬆️ 上传文件」通过文件选择器批量上传。

4. **一键级联删除（双端同步清理）**
   - 在文件卡片中点击「🗑️ 删除」，系统**同时在 Bucket A 与 Bucket B 中彻底删除对应文件**，无需两端反复手动确认。

5. **现代文件管理与可视化体验**
   - **图片实时预览**：自动为图片生成缩略图，点击支持全屏大图预览（私有读 Bucket 亦可通过后端安全直通预览）。
   - **双端状态徽章**：清晰呈现各文件一致性状态（🟢 双端一致 / 🔵 仅 A 端 / 🟣 仅 B 端 / 🟡 内容差异）。
   - **单文件对齐**：针对仅某端存在的文件，提供一键单文件对齐补齐按钮。
   - **子目录创建与面包屑**：支持在白名单目录下新建各级子目录，支持各级面包屑快速跳转导航。
   - **实时搜索**：支持在当前目录下按文件名秒级过滤检索。

6. **跨平台桌面客户端**
   - Windows 下单二进制运行，开箱即用独立桌面应用窗口（无 CMD 黑色控制台、无 Edge 微软账户弹窗打扰），内置专属炫酷高颜值图标。
   - 支持 macOS（Apple Silicon & Intel）与 Linux 纯 Go 零依赖运行。

---

## 🚀 快速使用

### 1. 运行桌面程序

- **Windows 桌面版**：直接双击 `release\aliyun-oss-sync.exe` 即可启动！
- **macOS Apple Silicon (M1/M2/M3/M4)**：`./release/aliyun-oss-sync-darwin-arm64`
- **macOS Intel (x86_64)**：`./release/aliyun-oss-sync-darwin-amd64`
- **Linux (x86_64)**：`./release/aliyun-oss-sync-linux-amd64`

### 2. 编译打包命令

- **Windows (GUI 纯桌面无黑框)**：
  ```bash
  go build -ldflags="-H windowsgui -s -w" -o release/aliyun-oss-sync.exe .
  ```
- **macOS Intel 芯片 (x86_64)**：
  ```bash
  CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-darwin-amd64 .
  ```
- **macOS Apple Silicon (M1/M2/M3/M4 arm64)**：
  ```bash
  CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-darwin-arm64 .
  ```
- **Linux (x86_64)**：
  ```bash
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o release/aliyun-oss-sync-linux-amd64 .
  ```
- **一键全平台打包脚本**：
  - Windows 环境双击执行：`scripts\build_all.bat`
  - macOS / Linux 环境执行：`./scripts/build_all.sh`

### 3. 使用步骤

1. **首次配置**：
   - 启动后自动弹出设置向导，输入 Bucket A 与 Bucket B 的凭证信息（支持一键复用 A 端凭据）。
   - 在下方添加需要允许管理的同步目录（例如 `images/`、`static/`）。
   - 点击「💾 保存并进入工作台」。
2. **进入目录**：
   - 页面根目录下将列出您配置的同步目录卡片，点击任意目录进入。
3. **上传/覆盖文件**：
   - 进入目录后，**直接从电脑文件夹中拖动文件释放到界面上**，即刻完成双端并发上传与覆盖！
4. **删除文件**：
   - 点击文件卡片下方的「🗑️」按钮确认删除，两端同名文件将被同步删除。
