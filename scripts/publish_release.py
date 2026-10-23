import os
import sys
import json
import urllib.request
import subprocess

TOKEN = os.environ.get("GITHUB_TOKEN")
if not TOKEN:
    TOKEN = input("请输入您的 GitHub Token (ghp_...): ").strip()

REPO = "yeskery/aliyun-oss-dir-sync-tools"
TAG = "v1.0.1"
RELEASE_NAME = "v1.0.1 阿里云 OSS 双端同步可视化工作台 (凭证/窗口/文件夹上传升级)"
PROXY = "http://127.0.0.1:10808"

# 1. Push git commits
print(f"🚀 [1/3] 正在推送代码至 GitHub 仓库 ({REPO})...")
remote_url = f"https://oauth2:{TOKEN}@github.com/{REPO}.git"
push_cmd = [
    "git", "-c", "credential.helper=",
    "-c", f"http.proxy={PROXY}",
    "push", remote_url, "main"
]
ret = subprocess.run(push_cmd, capture_output=True, text=True)
if ret.returncode != 0:
    print(f"Git push output: {ret.stderr or ret.stdout}")
else:
    print("✅ 代码成功推送至 GitHub main 分支！")

# Configure urllib with proxy
proxy_handler = urllib.request.ProxyHandler({
    'http': PROXY,
    'https': PROXY
})
opener = urllib.request.build_opener(proxy_handler)

# 2. Create Release
print(f"📦 [2/3] 正在创建 GitHub Release ({TAG})...")
release_body = """## 🌟 阿里云 OSS 双 Bucket 可视化文件管理工作台 v1.0.1

一款轻量、极速、无黑框的跨平台双端 OSS 对象存储管理客户端！

### 🚀 v1.0.1 更新内容
- **🎯 凭证复用优化**：点击「复用A凭据」时仅复制 AccessKeyID / AccessKeySecret，保留目标端原有的 Bucket 名称与 CDN 加速域名，不再误覆盖；
- **🖥️ 登录窗口防死锁与全屏居中**：修正主窗口关闭事件逻辑，窗口关闭后可随时再次点击登录按钮唤起，登录窗口在屏幕正中央显示；
- **🔴 凭证失效状态同步**：连接过期或失效时，顶部状态指示器清晰提示「🔴 凭据已失效 (点击重新登录)」，支持一键唤起登录框；
- **📁 支持直接上传文件夹**：修复文件夹上传时报「Incorrect bucket name or bucket name is too long」错误，原生递归上传文件夹及内部所有子文件并自动创建前缀。

### 📦 各平台下载解压即用
- **Windows 桌面版 (含橙色图标，纯GUI无黑框)**: `aliyun-oss-sync.exe`
- **macOS Apple Silicon 桌面版 (含橙色图标，解压得 .app)**: `Aliyun-OSS-Sync-macOS-arm64.zip`
- **macOS Intel 芯片桌面版 (含橙色图标，解压得 .app)**: `Aliyun-OSS-Sync-macOS-intel.zip`
- **Linux 64位桌面分发包 (含橙色图标与快捷方式)**: `Aliyun-OSS-Sync-linux-amd64.tar.gz`
- **Linux ARM64 桌面分发包 (含橙色图标与快捷方式)**: `Aliyun-OSS-Sync-linux-arm64.tar.gz`
- **macOS 命令行原生程序**: `aliyun-oss-sync-darwin-arm64` / `aliyun-oss-sync-darwin-amd64`
- **Linux 命令行原生程序**: `aliyun-oss-sync-linux-amd64` / `aliyun-oss-sync-linux-arm64`
"""

payload = json.dumps({
    "tag_name": TAG,
    "target_commitish": "main",
    "name": RELEASE_NAME,
    "body": release_body,
    "draft": False,
    "prerelease": False
}).encode("utf-8")

req = urllib.request.Request(
    f"https://api.github.com/repos/{REPO}/releases",
    data=payload,
    headers={
        "Authorization": f"token {TOKEN}",
        "Accept": "application/vnd.github.v3+json",
        "Content-Type": "application/json"
    },
    method="POST"
)

try:
    with opener.open(req) as resp:
        release_data = json.loads(resp.read().decode("utf-8"))
        release_id = release_data["id"]
        upload_url_base = release_data["upload_url"].split("{")[0]
        print(f"✅ Release 创建成功! Release ID: {release_id}")
except urllib.error.HTTPError as e:
    err_body = e.read().decode("utf-8")
    print(f"HTTP Error {e.code}: {err_body}")
    # If already exists, fetch it
    if e.code == 422:
        print("Release 已存在，获取现有 Release 信息...")
        req = urllib.request.Request(
            f"https://api.github.com/repos/{REPO}/releases/tags/{TAG}",
            headers={
                "Authorization": f"token {TOKEN}",
                "Accept": "application/vnd.github.v3+json"
            }
        )
        with opener.open(req) as resp:
            release_data = json.loads(resp.read().decode("utf-8"))
            upload_url_base = release_data["upload_url"].split("{")[0]
    else:
        sys.exit(1)

# 3. Upload Assets
print("📤 [3/3] 正在上传各平台可执行文件至 Releases...")
assets = [
    "release/aliyun-oss-sync.exe",
    "release/Aliyun-OSS-Sync-macOS-arm64.zip",
    "release/Aliyun-OSS-Sync-macOS-intel.zip",
    "release/Aliyun-OSS-Sync-linux-amd64.tar.gz",
    "release/Aliyun-OSS-Sync-linux-arm64.tar.gz",
    "release/aliyun-oss-sync-darwin-arm64",
    "release/aliyun-oss-sync-darwin-amd64",
    "release/aliyun-oss-sync-linux-amd64",
    "release/aliyun-oss-sync-linux-arm64"
]

for asset_path in assets:
    if not os.path.exists(asset_path):
        print(f"   ⚠️ 跳过不存在的文件: {asset_path}")
        continue

    file_name = os.path.basename(asset_path)
    file_size_mb = os.path.getsize(asset_path) / (1024 * 1024)
    print(f"   正在上传: {file_name} ({file_size_mb:.2f} MB)...", end="", flush=True)

    upload_url = f"{upload_url_base}?name={file_name}"
    with open(asset_path, "rb") as f:
        data = f.read()

    up_req = urllib.request.Request(
        upload_url,
        data=data,
        headers={
            "Authorization": f"token {TOKEN}",
            "Content-Type": "application/octet-stream"
        },
        method="POST"
    )

    try:
        with opener.open(up_req) as resp:
            print(" ✅ 上传成功!")
    except urllib.error.HTTPError as e:
        print(f" ❌ 上传失败: {e.code} {e.reason}")

# Reset origin url to clean token
subprocess.run(["git", "remote", "set-url", "origin", f"https://github.com/{REPO}.git"])

print("=" * 60)
print(f"🎉 v1.0.1 发布全部完成！")
print(f"   代码仓库: https://github.com/{REPO}")
print(f"   下载页面: https://github.com/{REPO}/releases/tag/{TAG}")
print("=" * 60)
